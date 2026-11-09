---
title: 文章标题
date: 2026-10-20
summary: 一两句话，出现在博客列表、首页和搜索结果里。
draft: true
---

把这个文件复制到 site/posts/<语言>/<网址名>.md，例如 site/posts/zh/netstream-timeout.md，
文章地址就是 /blog/zh/netstream-timeout/。写好后把上面的 draft: true 删掉再提交，就发布了。
（本文件放在 posts/ 下面一层，不会被当成文章。）

## 小标题

正文直接写，空一行是新段落。**加粗**、*斜体*、`代码`、[链接](https://github.com/githubflyideas/traffic66)。

- 列表
- 列表
  - 缩进两格是下一级

1. 编号
2. 编号

> 引用

```
ip netstream timeout active 1
```

| 项目 | 数值 |
|---|--:|
| 采样率 | 1:4096 |

图片：上传到和这篇 .md 同一个文件夹，然后这样写，引号里是图注（单独一行时会变成带说明的大图）：

![概览页](overview.png "图注写在这里")

网站自带的界面截图可以直接用：

![概览](assets/shots/zh/overview.jpg)
