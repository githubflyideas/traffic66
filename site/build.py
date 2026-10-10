#!/usr/bin/env python3
"""Builds the project website (GitHub Pages) into _site/.

Standard library only. Pages and posts are HTML fragments; this script adds
the shared header and footer, the blog index, the Atom feed and the
sitemap. A post is either site/posts/<lang>/<slug>.md (Markdown with its
title and date at the top, see site/README.md) or site/posts/<lang>/<slug>.html
listed in site/posts.json. Screenshots come from docs/images (assets/img on
the site).

    python3 site/build.py            # writes _site/
    python3 site/build.py --drafts   # also drafts and posts dated in the future
    python3 -m http.server -d _site  # look at it on http://localhost:8000/
"""
import datetime
import glob
import html
import json
import os
import re
import shutil
import sys

import mdlite

SITE = "https://githubflyideas.github.io/traffic66/"
HERE = os.path.dirname(os.path.abspath(__file__))
REPO = os.path.dirname(HERE)
OUT = os.path.join(REPO, "_site")
FONTS = ("https://fonts.googleapis.com/css2?family=Source+Serif+4:opsz,wght@8..60,600;8..60,800"
         "&family=IBM+Plex+Sans:wght@400;500;600;700&family=JetBrains+Mono:wght@400;600&display=swap")
MARK = ('<svg viewBox="0 0 56 56" aria-hidden="true"><rect width="56" height="56" rx="12" fill="#2a78d6"/>'
        '<path d="M10 40l12-16 10 8 14-20" stroke="#fff" stroke-width="5" stroke-linecap="round" '
        'stroke-linejoin="round" fill="none"/><circle cx="46" cy="12" r="4" fill="#fff"/></svg>')
NAV = [("home", "", "Home"), ("blog", "blog/", "Blog"),
       ("", "https://github.com/githubflyideas/traffic66/releases", "Download"),
       ("", "https://github.com/githubflyideas/traffic66", "GitHub")]
# html lang and text direction of each post language
LANGS = {"ar": "rtl", "ur": "rtl"}
# fonts for scripts the Latin faces do not cover: (Google Fonts families, display stack, body stack)
SCRIPT_FONTS = {
    "zh": ("Noto+Serif+SC:wght@600;900&family=Noto+Sans+SC:wght@400;500;700",
           '"Noto Serif SC","Songti SC",serif', '"Noto Sans SC","PingFang SC","Microsoft YaHei",sans-serif'),
    "ja": ("Noto+Serif+JP:wght@600;900&family=Noto+Sans+JP:wght@400;500;700",
           '"Noto Serif JP","Hiragino Mincho ProN",serif', '"Noto Sans JP","Hiragino Sans","Yu Gothic",sans-serif'),
    "ko": ("Noto+Serif+KR:wght@600;900&family=Noto+Sans+KR:wght@400;500;700",
           '"Noto Serif KR",serif', '"Noto Sans KR","Apple SD Gothic Neo","Malgun Gothic",sans-serif'),
    "ar": ("Noto+Naskh+Arabic:wght@600;700&family=IBM+Plex+Sans+Arabic:wght@400;500;600;700",
           '"Noto Naskh Arabic",serif', '"IBM Plex Sans Arabic",Tahoma,sans-serif'),
    "ur": ("Noto+Nastaliq+Urdu:wght@400;700",
           '"Noto Nastaliq Urdu","Jameel Noori Nastaleeq",serif', '"Noto Nastaliq Urdu","Jameel Noori Nastaleeq",serif'),
    "hi": ("Noto+Serif+Devanagari:wght@600;800&family=Noto+Sans+Devanagari:wght@400;500;700",
           '"Noto Serif Devanagari",serif', '"Noto Sans Devanagari","Mangal",sans-serif'),
    "bn": ("Noto+Serif+Bengali:wght@600;800&family=Noto+Sans+Bengali:wght@400;500;700",
           '"Noto Serif Bengali",serif', '"Noto Sans Bengali","Vrinda",sans-serif'),
}


def read(*p):
    with open(os.path.join(HERE, *p), encoding="utf-8") as f:
        return f.read()


def write(rel, text):
    path = os.path.join(OUT, rel)
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w", encoding="utf-8") as f:
        f.write(text)


def page(rel, title, desc, body, cur="", image="assets/img/overview.png", kind="website", lang="en", root=None):
    """A whole page at rel (e.g. "blog/index.html") around a body fragment.
    {root} in the fragment becomes the relative way back to the site root."""
    if root is None:
        root = "../" * rel.count("/")
    canon = SITE + rel.removesuffix("index.html")
    current = ' aria-current="page"'
    nav = "".join(
        f'<a href="{h if h.startswith("http") else root + h}"{current if k and k == cur else ""}>{t}</a>'
        for k, h, t in NAV)
    d = LANGS.get(lang, "ltr")
    e = html.escape
    extra = ""
    if lang in SCRIPT_FONTS:
        fam, disp, bod = SCRIPT_FONTS[lang]
        extra = (f'<link rel="stylesheet" href="https://fonts.googleapis.com/css2?family={fam}&display=swap">\n'
                 f'<style>:root{{--display:{disp};--body:{bod}}}</style>\n')
    return f"""<!doctype html>
<html lang="{lang}" dir="{d}">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{e(title)}</title>
<meta name="description" content="{e(desc)}">
<link rel="canonical" href="{canon}">
<meta property="og:type" content="{kind}">
<meta property="og:title" content="{e(title)}">
<meta property="og:description" content="{e(desc)}">
<meta property="og:url" content="{canon}">
<meta property="og:image" content="{SITE}{image}">
<meta name="twitter:card" content="summary_large_image">
<link rel="icon" href="{root}assets/logo-mark.svg" type="image/svg+xml">
<link rel="alternate" type="application/atom+xml" title="traffic66 blog" href="{root}blog/feed.xml">
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link rel="stylesheet" href="{FONTS}">
<link rel="stylesheet" href="{root}assets/site.css">
{extra}</head>
<body>
<header class="site-head"><div class="in"><a class="brand" href="{root or './'}">{MARK}<span>traffic<b>66</b></span></a><nav aria-label="Site">{nav}</nav></div></header>
{body.replace("{root}", root)}
<footer class="site-foot"><span>traffic66 · flow analytics for sFlow, NetFlow and IPFIX</span><span><a href="https://github.com/githubflyideas/traffic66/blob/main/LICENSE.md">Licence</a> · <a href="https://github.com/githubflyideas/traffic66/issues">Issues</a> · <a href="{root}blog/feed.xml">Feed</a></span></footer>
</body>
</html>
"""


def md_posts():
    """Posts written as site/posts/<lang>/<slug>.md, with their front matter."""
    out = []
    for path in sorted(glob.glob(os.path.join(HERE, "posts", "*", "*.md"))):
        lang = os.path.basename(os.path.dirname(path))
        rel = os.path.relpath(path, REPO)
        with open(path, encoding="utf-8") as f:
            meta, body = mdlite.front_matter(f.read())
        lines = body.replace("\r\n", "\n").split("\n")
        title = meta.get("title")
        if not title:
            # no title: line, so the first "# heading" is the title
            for k, line in enumerate(lines):
                if line.strip():
                    if line.startswith("# "):
                        title = line[2:].strip()
                        del lines[k]
                    break
        if not title:
            sys.exit(f"{rel}: no title (put 'title: ...' at the top, or start with '# Title')")
        date = str(meta.get("date", ""))
        try:
            datetime.date.fromisoformat(date)
        except ValueError:
            sys.exit(f"{rel}: date must look like 2026-10-08, got {date!r}")
        body = "\n".join(lines)
        summary = meta.get("summary")
        if not summary:
            # the first paragraph, as plain text
            first = re.split(r"\n\s*\n", body.strip(), maxsplit=1)[0]
            summary = re.sub(r"<[^>]+>", "", mdlite.inline(first))
            summary = html.unescape(re.sub(r"\s+", " ", summary)).strip()
            if len(summary) > 160:
                summary = summary[:157].rstrip() + "…"
        p = {"date": date, "lang": lang, "slug": meta.get("slug") or os.path.basename(path)[:-3],
             "title": title, "summary": summary, "md": body, "dir": os.path.dirname(path), "src": rel}
        for k in ("image", "author", "lede"):
            if meta.get(k):
                p[k] = meta[k]
        if meta.get("draft") is True:
            p["draft"] = True
        out.append(p)
    return out


def md_body(p, outdir):
    """The HTML of a Markdown post. Pictures next to the .md are copied
    beside the page; links starting with assets/ point at the site's assets."""
    def link(u):
        if re.match(r"^([a-z][a-z0-9+.-]*:|/|#|\{root\})", u, re.I):
            return u
        if u.startswith("assets/"):
            return "{root}" + u
        name = u.split("#")[0].split("?")[0]
        src = os.path.normpath(os.path.join(p["dir"], name))
        if name and os.path.isfile(src) and src.startswith(p["dir"] + os.sep):
            dst = os.path.join(outdir, os.path.relpath(src, p["dir"]))
            os.makedirs(os.path.dirname(dst), exist_ok=True)
            shutil.copyfile(src, dst)
        return u
    e = html.escape
    by = f'<div class="byline"><span>{e(p["author"])}</span></div>' if p.get("author") else ""
    return (f'<article>\n<header class="hero col">\n'
            f'<div class="kicker"><span><a href="{{root}}blog/" style="color:inherit">traffic66 blog</a></span>'
            f'<b>{e(p["date"])}</b></div>\n<h1>{e(p["title"])}</h1>\n'
            f'<p class="lede">{e(p.get("lede") or p["summary"])}</p>\n{by}</header>\n'
            f'<div class="col md">\n{mdlite.to_html(p["md"], link)}\n</div>\n</article>\n')


def post_list(posts, root):
    e = html.escape
    items = "".join(
        f'<li><time datetime="{p["date"]}">{p["date"]}</time><div><a href="{root}blog/{p["lang"]}/{p["slug"]}/" lang="{p["lang"]}">{e(p["title"])}</a>'
        f'<span class="lang">{p["lang"].upper()}</span><p lang="{p["lang"]}">{e(p["summary"])}</p></div></li>'
        for p in posts)
    return f'<ul class="posts">{items}</ul>'


def main():
    # a post with "draft": true is left out until it is published
    drafts = "--drafts" in sys.argv
    posts = json.loads(read("posts.json")) + md_posts()
    seen = {}
    for p in posts:
        key = (p["lang"], p["slug"])
        if key in seen:
            sys.exit(f"two posts at blog/{p['lang']}/{p['slug']}/: {seen[key]} and {p.get('src', 'posts.json')}")
        seen[key] = p.get("src", "posts.json")
    # a post dated after today (Tokyo time) waits for that day: the site is
    # rebuilt every morning, so dated posts go out one a day by themselves
    today = (datetime.datetime.now(datetime.timezone.utc) + datetime.timedelta(hours=9)).date().isoformat()
    posts = [p for p in posts if drafts or (not p.get("draft") and p["date"] <= today)]
    posts.sort(key=lambda p: p["date"], reverse=True)
    shutil.rmtree(OUT, ignore_errors=True)
    shutil.copytree(os.path.join(HERE, "assets"), os.path.join(OUT, "assets"))
    shutil.copytree(os.path.join(REPO, "docs", "images"), os.path.join(OUT, "assets", "img"))

    write("index.html", page("index.html", "traffic66 · NetFlow, sFlow and IPFIX analyzer in one program",
          "Flow analytics for sFlow, NetFlow and IPFIX in a single program for Windows, Linux and macOS: bandwidth, "
          "top talkers, interface counter check, detection and pcap analysis.",
          read("pages", "home.html").replace("{posts}", post_list(posts[:5], "{root}")), "home"))
    write("blog/index.html", page("blog/index.html", "Blog · traffic66",
          "Articles about flow analytics with sFlow, NetFlow and IPFIX, and about traffic66.",
          read("pages", "blog.html").replace("{posts}", post_list(posts, "{root}")), "blog"))
    # served for any missing path, so links start from the site root
    write("404.html", page("404.html", "Page not found · traffic66", "The page was not found.",
          read("pages", "404.html"), root="/traffic66/"))
    for p in posts:
        rel = f'blog/{p["lang"]}/{p["slug"]}/index.html'
        body = (md_body(p, os.path.dirname(os.path.join(OUT, rel))) if "md" in p
                else read("posts", p["lang"], p["slug"] + ".html"))
        write(rel, page(rel, p["title"] + " · traffic66", p["summary"], body,
                        "blog", p.get("image", "assets/img/overview.png"), "article", p["lang"]))

    e = html.escape
    entries = "".join(
        f'<entry><title>{e(p["title"])}</title><link href="{SITE}blog/{p["lang"]}/{p["slug"]}/"/>'
        f'<id>{SITE}blog/{p["lang"]}/{p["slug"]}/</id><updated>{p["date"]}T00:00:00Z</updated>'
        f'<summary>{e(p["summary"])}</summary><author><name>traffic66 developers</name></author></entry>'
        for p in posts)
    updated = posts[0]["date"] if posts else "2026-01-01"
    write("blog/feed.xml", f'<?xml version="1.0" encoding="utf-8"?>\n<feed xmlns="http://www.w3.org/2005/Atom">'
          f'<title>traffic66 blog</title><link href="{SITE}blog/"/><link rel="self" href="{SITE}blog/feed.xml"/>'
          f'<id>{SITE}blog/</id><updated>{updated}T00:00:00Z</updated>{entries}</feed>\n')
    urls = [("", updated), ("blog/", updated)] + [(f'blog/{p["lang"]}/{p["slug"]}/', p["date"]) for p in posts]
    write("sitemap.xml", '<?xml version="1.0" encoding="UTF-8"?>\n<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">\n'
          + "".join(f"<url><loc>{SITE}{u}</loc><lastmod>{d}</lastmod></url>\n" for u, d in urls) + "</urlset>\n")
    write("robots.txt", f"User-agent: *\nAllow: /\nSitemap: {SITE}sitemap.xml\n")
    write(".nojekyll", "")
    print(f"built {OUT}: {len(posts)} posts" + (" (drafts included)" if drafts else ""))


if __name__ == "__main__":
    main()
