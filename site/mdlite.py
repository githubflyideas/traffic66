"""A small Markdown to HTML converter for the blog (standard library only).

Covers what a post needs: front matter, headings, paragraphs, line breaks,
bold, italic, strikethrough, inline code, links, images (a picture alone in
a paragraph becomes a figure with its caption), bulleted, numbered and
nested lists, task lists, quotes, code blocks, tables and rules. A block that
starts with an HTML tag is passed through unchanged, so anything the
Markdown cannot say can still be written in HTML.
"""
import html
import re

FRONT = re.compile(r"\A---[ \t]*\r?\n(.*?)\r?\n---[ \t]*(?:\r?\n|\Z)", re.S)


def front_matter(text):
    """Splits "---\\nkey: value\\n---\\nbody" into ({key: value}, body)."""
    m = FRONT.match(text.lstrip("﻿"))
    if not m:
        return {}, text
    meta = {}
    for line in m.group(1).splitlines():
        if not line.strip() or line.lstrip().startswith("#") or ":" not in line:
            continue
        k, v = line.split(":", 1)
        v = v.strip()
        if len(v) >= 2 and v[0] == v[-1] and v[0] in "\"'":
            v = v[1:-1]
        low = v.lower()
        meta[k.strip().lower()] = True if low in ("true", "yes") else False if low in ("false", "no") else v
    return meta, text[m.end():]


# ---- inline -------------------------------------------------------------

def _url(u, link):
    """A URL from the post; link(u) can rewrite relative ones."""
    u = u.strip()
    if u.startswith("<") and u.endswith(">"):
        u = u[1:-1]
    if re.match(r"(?i)\s*(javascript|vbscript|data):", u) and not u.lower().startswith("data:image/"):
        return "#"
    return html.escape(link(u) if link else u, quote=True)


_LINK = r"\[((?:[^\[\]]|\[[^\]]*\])*)\]\(\s*(<[^>]*>|[^\s)]+)(?:\s+\"([^\"]*)\")?\s*\)"
_TOKENS = re.compile(
    r"(?P<code>`+)(?P<ctext>.+?)(?P=code)"
    r"|(?P<img>!" + _LINK + r")"
    r"|(?P<link>" + _LINK + r")"
    r"|<(?P<auto>https?://[^\s>]+)>"
    r"|(?P<tag></?[A-Za-z][A-Za-z0-9-]*(?:\s[^<>]*)?/?>)"
    r"|(?P<esc>\\[\\`*_{}\[\]()#+\-.!|~<>])"
    r"|(?P<br> {2,}\n|\\\n)",
    re.S)


def inline(s, link=None):
    # tokens become placeholders so that emphasis can span them (**[a](b)**)
    # but never reaches inside them (underscores in a URL)
    out, held, pos = [], [], 0

    def hold(h):
        held.append(h)
        return f"\x00{len(held) - 1}\x00"
    for m in _TOKENS.finditer(s):
        out.append(html.escape(s[pos:m.start()].replace("\x00", ""), quote=False))
        pos = m.end()
        g = m.group
        n = len(out)
        if g("code"):
            out.append("<code>" + html.escape(g("ctext").strip(), quote=False) + "</code>")
        elif g("img"):
            alt, src, title = re.match(r"!" + _LINK, g("img")).groups()
            t = f' title="{html.escape(title)}"' if title else ""
            out.append(f'<img src="{_url(src, link)}" alt="{html.escape(alt)}"{t} loading="lazy">')
        elif g("link"):
            text, href, title = re.match(_LINK, g("link")).groups()
            t = f' title="{html.escape(title)}"' if title else ""
            out.append(f'<a href="{_url(href, link)}"{t}>{inline(text, link)}</a>')
        elif g("auto"):
            u = html.escape(g("auto"))
            out.append(f'<a href="{u}">{u}</a>')
        elif g("tag"):
            out.append(g("tag"))
        elif g("esc"):
            out.append(html.escape(g("esc")[1], quote=False))
        elif g("br"):
            out.append("<br>\n")
        out[n:] = [hold("".join(out[n:]))]
    out.append(html.escape(s[pos:].replace("\x00", ""), quote=False))
    return re.sub(r"\x00(\d+)\x00", lambda m: held[int(m.group(1))], _emph("".join(out)))


def _emph(s):
    s = re.sub(r"\*\*(?=\S)(.+?)(?<=\S)\*\*", r"<strong>\1</strong>", s)
    s = re.sub(r"(?<![\w\\])__(?=\S)(.+?)(?<=\S)__(?!\w)", r"<strong>\1</strong>", s)
    s = re.sub(r"~~(?=\S)(.+?)(?<=\S)~~", r"<del>\1</del>", s)
    s = re.sub(r"\*(?=[^\s*])(.+?)(?<=[^\s*])\*", r"<em>\1</em>", s)
    s = re.sub(r"(?<![\w\\])_(?=[^\s_])(.+?)(?<=[^\s_])_(?!\w)", r"<em>\1</em>", s)
    return s


# ---- blocks -------------------------------------------------------------

_HR = re.compile(r"^ {0,3}([-*_])(?:\s*\1){2,}\s*$")
_HEAD = re.compile(r"^ {0,3}(#{1,6})\s+(.*?)\s*#*\s*$")
_FENCE = re.compile(r"^ {0,3}(`{3,}|~{3,})\s*([\w+#.-]*)")
_ITEM = re.compile(r"^( {0,3})([-*+]|\d{1,9}[.)])\s+(.*)$")
_HTML = re.compile(r"^ {0,3}<(/?[A-Za-z][A-Za-z0-9-]*|!--)")
_TROW = re.compile(r"^\s*\|?.*\|.*$")
_TSEP = re.compile(r"^\s*\|?\s*:?-{2,}:?\s*(\|\s*:?-{2,}:?\s*)*\|?\s*$")


def _cells(row):
    row = row.strip()
    if row.startswith("|"):
        row = row[1:]
    if row.endswith("|") and not row.endswith("\\|"):
        row = row[:-1]
    return [c.strip().replace("\\|", "|") for c in re.split(r"(?<!\\)\|", row)]


def blocks(lines, link=None, top=False):
    out, i, n = [], 0, len(lines)

    def starts_block(line):
        return (not line.strip() or _HR.match(line) or _HEAD.match(line) or _FENCE.match(line)
                or line.lstrip().startswith(">") or _ITEM.match(line) or _HTML.match(line))

    while i < n:
        line = lines[i]
        if not line.strip():
            i += 1
            continue
        m = _FENCE.match(line)
        if m:
            mark, lang = m.group(1), m.group(2)
            j, code = i + 1, []
            while j < n and not lines[j].strip().startswith(mark):
                code.append(lines[j])
                j += 1
            cls = f' class="language-{html.escape(lang)}"' if lang else ""
            out.append(f"<pre><code{cls}>" + html.escape("\n".join(code), quote=False) + "</code></pre>")
            i = j + 1
            continue
        m = _HEAD.match(line)
        if m:
            # one # is the post's own title, so headings in the body start at h2
            level = min(max(len(m.group(1)), 2), 6)
            text = inline(m.group(2), link)
            slug = re.sub(r"[^\w-]+", "-", re.sub(r"<[^>]+>", "", m.group(2)).lower()).strip("-")
            ident = f' id="{html.escape(slug)}"' if slug else ""
            out.append(f"<h{level}{ident}>{text}</h{level}>")
            i += 1
            continue
        if _HR.match(line):
            out.append("<hr>")
            i += 1
            continue
        if _HTML.match(line):
            j = i
            while j < n and lines[j].strip():
                j += 1
            out.append("\n".join(lines[i:j]))
            i = j
            continue
        if line.lstrip().startswith(">"):
            j, inner = i, []
            while j < n and lines[j].strip() and (lines[j].lstrip().startswith(">") or not starts_block(lines[j])):
                inner.append(re.sub(r"^\s*>\s?", "", lines[j]))
                j += 1
            out.append("<blockquote>\n" + blocks(inner, link) + "\n</blockquote>")
            i = j
            continue
        m = _ITEM.match(line)
        if m:
            ordered = m.group(2)[0].isdigit()
            base = len(m.group(1))
            items, j = [], i
            while j < n:
                mm = _ITEM.match(lines[j])
                if mm and len(mm.group(1)) == base and mm.group(2)[0].isdigit() == ordered:
                    items.append([mm.group(3)])
                    j += 1
                    continue
                if not lines[j].strip():
                    # a blank line ends the list unless the next line carries on with it
                    k = j + 1
                    if k < n and (lines[k].startswith(" " * (base + 2)) or
                                  (_ITEM.match(lines[k]) and len(_ITEM.match(lines[k]).group(1)) == base)):
                        items[-1].append("")
                        j += 1
                        continue
                    break
                if lines[j].startswith(" " * (base + 1)) or not starts_block(lines[j]):
                    items[-1].append(lines[j][base + 2:] if lines[j].startswith(" " * (base + 2)) else lines[j].strip())
                    j += 1
                    continue
                break
            start = ""
            if ordered:
                first = int(re.match(r"\d+", m.group(2)).group())
                start = f' start="{first}"' if first != 1 else ""
            lis = []
            for it in items:
                body = blocks(it, link)
                if body.startswith("<p>") and body.count("<p>") == 1:
                    body = body[3:].replace("</p>", "", 1)
                tm = re.match(r"\[([ xX])\]\s+", body)
                if tm:
                    box = "checked " if tm.group(1) != " " else ""
                    body = f'<input type="checkbox" disabled {box}aria-hidden="true"> ' + body[tm.end():]
                lis.append(f"<li>{body}</li>")
            tag = "ol" if ordered else "ul"
            out.append(f"<{tag}{start}>" + "".join(lis) + f"</{tag}>")
            i = j
            continue
        if _TROW.match(line) and i + 1 < n and _TSEP.match(lines[i + 1]):
            head = _cells(line)
            aligns = []
            for c in _cells(lines[i + 1]):
                aligns.append("center" if c.startswith(":") and c.endswith(":") else
                              "right" if c.endswith(":") else "left" if c.startswith(":") else "")
            j, rows = i + 2, []
            while j < n and lines[j].strip() and "|" in lines[j]:
                rows.append(_cells(lines[j]))
                j += 1

            def cell(t, c, k):
                a = aligns[k] if k < len(aligns) and aligns[k] else ""
                st = f' style="text-align:{a}"' if a else ""
                return f"<{t}{st}>{inline(c, link)}</{t}>"
            th = "".join(cell("th", c, k) for k, c in enumerate(head))
            trs = "".join("<tr>" + "".join(cell("td", r[k] if k < len(r) else "", k) for k in range(len(head))) + "</tr>"
                          for r in rows)
            out.append(f'<div class="tablewrap md"><table><thead><tr>{th}</tr></thead><tbody>{trs}</tbody></table></div>')
            i = j
            continue
        # paragraph
        j = i
        while j < n and lines[j].strip() and (j == i or not starts_block(lines[j])):
            if j > i and _TROW.match(lines[j]) and j + 1 < n and _TSEP.match(lines[j + 1]):
                break
            j += 1
        text = "\n".join(lines[i:j])
        # a line break between two Chinese or Japanese characters is not a space
        text = re.sub(r"(?<=[\u2e80-\u9fff\uf900-\ufaff\uff00-\uffef])\n(?=[\u2e80-\u9fff\uf900-\ufaff\uff00-\uffef])", "", text)
        i = j
        m = re.fullmatch(r"\s*!" + _LINK + r"\s*", text, re.S)
        if m:
            alt, src, title = m.groups()
            cap = title or alt
            fig = f'<img src="{_url(src, link)}" alt="{html.escape(alt)}" loading="lazy">'
            if cap:
                fig += f"<figcaption>{inline(cap, link)}</figcaption>"
            out.append(f'<figure class="shot">{fig}</figure>')
            continue
        out.append("<p>" + inline(text.strip(), link) + "</p>")
    return "\n".join(out)


def to_html(text, link=None):
    """Markdown body to HTML. link(url) may rewrite link and image URLs."""
    text = text.replace("\r\n", "\n").replace("\t", "    ")
    return blocks(text.split("\n"), link)
