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

function upgradeMermaidBlocks() {
  const mermaidCodeBlocks = document.querySelectorAll(
    ".markdown-body pre > code.language-mermaid, .markdown-body pre > code.lang-mermaid"
  );

  for (const codeBlock of mermaidCodeBlocks) {
    const pre = codeBlock.parentElement;
    if (!pre || !pre.parentElement) {
      continue;
    }

    const mermaidContainer = document.createElement("div");
    mermaidContainer.className = "mermaid";
    mermaidContainer.textContent = codeBlock.textContent.trim();
    pre.replaceWith(mermaidContainer);
  }
}

async function renderMermaidDiagrams() {
  upgradeMermaidBlocks();

  if (!document.querySelector(".markdown-body .mermaid")) {
    return;
  }

  const mermaid = await import("https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.esm.min.mjs");
  mermaid.default.initialize({
    startOnLoad: false,
    theme: "base",
    themeVariables: {
      primaryColor: "#ffffff",
      primaryTextColor: "#151515",
      primaryBorderColor: "#c7c7c7",
      lineColor: "#ee0000",
      secondaryColor: "#f5f5f5",
      tertiaryColor: "#fffaf7",
      fontFamily: "Red Hat Text, Helvetica Neue, Arial, sans-serif",
    },
    flowchart: {
      curve: "linear",
      htmlLabels: true,
    },
  });
  await mermaid.default.run({
    querySelector: ".markdown-body .mermaid",
  });
}

const COPY_LABELS = { idle: 'Copy', copied: 'Copied', failed: 'Failed' };

async function copyCodeboxCode(button) {
  const codebox = button.closest('.codebox');
  const code = codebox?.querySelector('pre code');
  const label = button.querySelector('.codebox__copy-label');
  if (!code || !label) return;
  try {
    await navigator.clipboard.writeText(code.textContent ?? '');
    button.dataset.copyState = 'copied';
  } catch {
    button.dataset.copyState = 'failed';
  }
  label.textContent = COPY_LABELS[button.dataset.copyState] ?? COPY_LABELS.idle;
  window.setTimeout(() => {
    button.dataset.copyState = 'idle';
    label.textContent = COPY_LABELS.idle;
  }, 1800);
}

function toggleCodeboxWrap(button) {
  const codebox = button.closest('.codebox');
  if (!codebox) return;
  const nextWrapped = !codebox.classList.contains('codebox--wrapped');
  codebox.classList.toggle('codebox--wrapped', nextWrapped);
  button.setAttribute('aria-pressed', nextWrapped ? 'true' : 'false');
}

document.addEventListener('click', (event) => {
  const target = event.target instanceof Element ? event.target : null;
  const copyButton = target?.closest('.codebox__copy');
  if (copyButton instanceof HTMLButtonElement) {
    void copyCodeboxCode(copyButton);
    return;
  }
  const wrapButton = target?.closest('.codebox__wrap');
  if (wrapButton instanceof HTMLButtonElement) {
    toggleCodeboxWrap(wrapButton);
  }
});

async function initializeDocs() {
  buildPageToc();
  await renderMermaidDiagrams();
}

if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", () => {
    void initializeDocs();
  }, { once: true });
} else {
  void initializeDocs();
}
