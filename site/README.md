# Project website

The site at https://githubflyideas.github.io/traffic66/ is built from this
folder by `site/build.py` (Python standard library only) and published by
`.github/workflows/pages.yml` on every push to `main` that touches `site/`
or `docs/images/`.

```
site/
  build.py              builds _site/ (header, footer, blog index, feed, sitemap)
  posts.json            the list of posts: date, lang, slug, title, summary, image
  posts/<lang>/<slug>.md     a post in Markdown (title and date at the top)
  posts/<lang>/<slug>.html   a post's body as an HTML fragment (listed in posts.json)
  posts/TEMPLATE.md     a Markdown post to copy
  mdlite.py             the Markdown converter
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

### In Markdown (easiest; works entirely in the GitHub web page)

1. On GitHub open `site/posts/<lang>/` (for example `site/posts/zh/`) and
   click **Add file → Create new file**.
2. Name it `<slug>.md`, for example `netstream-timeout.md`; the post will be
   at `/blog/zh/netstream-timeout/`. Use lowercase letters, digits and `-`.
3. Start it with the title and date, then write:

   ```
   ---
   title: 文章标题
   date: 2026-10-20
   summary: 一两句话，出现在博客列表和搜索结果里（可省略，默认取第一段）
   draft: true
   ---

   正文……
   ```

   `site/posts/TEMPLATE.md` shows everything: headings, lists, quotes,
   code, tables, links and pictures. The **Preview** tab in the GitHub
   editor shows roughly how it will look.
4. Pictures: in the same folder, **Add file → Upload files**, then
   `![alt text](picture.png "caption")` on a line of its own. The UI
   screenshots on the site can be used directly:
   `![Overview](assets/shots/zh/overview.jpg)`.
5. **Commit changes**. With `draft: true` it is saved but not shown; delete
   that line (and set `date`) to publish. The site updates in a minute or two.

Optional front matter: `image:` (the picture shown when the link is shared,
default the overview screenshot), `author:`, `lede:` (text under the title,
default the summary), `slug:` (a different address than the file name).
If the build stops, the Actions page says which file and why.

HTML still works in Markdown: a block that starts with a tag is kept as is.

### In HTML (for posts with special layout)

1. Write `site/posts/<lang>/<slug>.html` (copy an existing post for the
   markup: `hero`, `col`, `wide`, `tablewrap`, `vs`, `grid2`, `pick`).
2. Add an entry at the top of `posts.json`.

Either way: look at it with `python3 site/build.py && python3 -m http.server -d _site`.

## Drafts and serial publishing

A post with `"draft": true` in `posts.json`, or `draft: true` at the top of
a Markdown post, is in the repository but not on the site. To publish it,
remove the draft mark and set the date to the day it goes out. `python3 site/build.py --drafts` builds a preview
with the drafts included.

Screenshots of the traffic66 UI in each language are in
`assets/shots/<lang>/` (overview, interfaces, findings).

The post appears at `/blog/<lang>/<slug>/`, in the blog index, on the home
page (the five newest), in `blog/feed.xml` and in `sitemap.xml`.
