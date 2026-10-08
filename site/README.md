# Project website

The site at https://githubflyideas.github.io/traffic66/ is built from this
folder by `site/build.py` (Python standard library only) and published by
`.github/workflows/pages.yml` on every push to `main` that touches `site/`
or `docs/images/`.

```
site/
  build.py              builds _site/ (header, footer, blog index, feed, sitemap)
  posts.json            the list of posts: date, lang, slug, title, summary, image
  posts/<lang>/<slug>.html   one post's body (HTML fragment)
  pages/home.html       home page body
  pages/blog.html       blog index body ({posts} is the list)
  pages/404.html        not-found page body
  assets/site.css       styles shared by every page (light and dark)
  assets/logo-mark.svg  icon
docs/images/            screenshots, published as assets/img/
```

In fragments, `{root}` is the relative way back to the site root, so links
and images work at any depth: `<img src="{root}assets/img/overview.png">`.

## Add a post

1. Write `site/posts/<lang>/<slug>.html` (copy an existing post for the
   markup: `hero`, `col`, `wide`, `tablewrap`, `vs`, `grid2`, `pick`).
2. Add an entry at the top of `posts.json`.
3. Look at it: `python3 site/build.py && python3 -m http.server -d _site`.
4. Merge to `main`; the site updates within a minute or two.

Posts in Arabic and Urdu are set right to left automatically; Chinese,
Japanese, Korean and Arabic posts get fonts for their script.

## Drafts and serial publishing

A post with `"draft": true` in `posts.json` is in the repository but not on
the site. To publish it, remove `"draft": true` and set `"date"` to the day
it goes out, then merge. `python3 site/build.py --drafts` builds a preview
with the drafts included.

Screenshots of the traffic66 UI in each language are in
`assets/shots/<lang>/` (overview, interfaces, findings).

The post appears at `/blog/<lang>/<slug>/`, in the blog index, on the home
page (the five newest), in `blog/feed.xml` and in `sitemap.xml`.
