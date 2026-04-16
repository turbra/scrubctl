function slugifyHeading(text) {
  return text
    .toLowerCase()
    .trim()
    .replace(/[^\w\s-]/g, "")
    .replace(/\s+/g, "-");
}

function buildPageToc() {
  const tocRoot = document.getElementById("sc-toc-list");
  const tocWrapper = document.getElementById("sc-page-toc");
  if (!tocRoot || !tocWrapper) {
    return;
  }

  const headings = [...document.querySelectorAll(".markdown-body h2")];
  if (!headings.length) {
    return;
  }

  const usedIds = new Set([...document.querySelectorAll("[id]")].map((element) => element.id));

  for (const heading of headings) {
    if (!heading.id) {
      const baseId = slugifyHeading(heading.textContent);
      let nextId = baseId;
      let suffix = 2;

      while (usedIds.has(nextId)) {
        nextId = `${baseId}-${suffix}`;
        suffix += 1;
      }

      heading.id = nextId;
      usedIds.add(nextId);
    }

    const item = document.createElement("li");
    const link = document.createElement("a");
    link.href = `#${heading.id}`;
    link.textContent = heading.textContent.trim();
    item.appendChild(link);
    tocRoot.appendChild(item);
  }

  tocWrapper.hidden = false;
}

if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", buildPageToc, { once: true });
} else {
  buildPageToc();
}
