#!/usr/bin/env python3
"""Builds the project website (GitHub Pages) into _site/.

Standard library only. Pages and posts are HTML fragments; this script adds
the shared header and footer, the blog index, the Atom feed and the
sitemap. To add a post: write site/posts/<lang>/<slug>.html and add it to
site/posts.json. Screenshots come from docs/images (assets/img on the site).

    python3 site/build.py            # writes _site/
    python3 site/build.py --drafts   # also the posts marked "draft": true
    python3 -m http.server -d _site  # look at it on http://localhost:8000/
"""
import html
import json
import os
import shutil
import sys

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
    posts = [p for p in json.loads(read("posts.json")) if drafts or not p.get("draft")]
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
        write(rel, page(rel, p["title"] + " · traffic66", p["summary"], read("posts", p["lang"], p["slug"] + ".html"),
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
