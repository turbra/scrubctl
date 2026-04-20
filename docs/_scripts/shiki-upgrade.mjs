import { codeToHtml } from 'shiki';
import { load } from 'cheerio';
import { readFileSync, writeFileSync, readdirSync, statSync } from 'fs';
import { join, extname } from 'path';

const THEME = 'github-dark-high-contrast';

const LANGUAGE_LABELS = {
  bash: 'Shell', sh: 'Shell', shell: 'Shell', console: 'Shell',
  javascript: 'JavaScript', js: 'JavaScript',
  typescript: 'TypeScript', ts: 'TypeScript',
  json: 'JSON', yaml: 'YAML', yml: 'YAML',
  python: 'Python', py: 'Python',
  html: 'HTML', xml: 'XML', toml: 'TOML', text: 'Text',
  go: 'Go', ruby: 'Ruby', rust: 'Rust',
};

const COPY_ICON = `<svg viewBox="0 0 16 16" focusable="false"><path d="M5 1.75A1.75 1.75 0 0 1 6.75 0h5.5A1.75 1.75 0 0 1 14 1.75v7.5A1.75 1.75 0 0 1 12.25 11h-5.5A1.75 1.75 0 0 1 5 9.25zm1.75-.25a.25.25 0 0 0-.25.25v7.5c0 .138.112.25.25.25h5.5a.25.25 0 0 0 .25-.25v-7.5a.25.25 0 0 0-.25-.25z"></path><path d="M2.75 4A1.75 1.75 0 0 0 1 5.75v7.5C1 14.217 1.784 15 2.75 15h5.5A1.75 1.75 0 0 0 10 13.25V13H8.5v.25a.25.25 0 0 1-.25.25h-5.5a.25.25 0 0 1-.25-.25v-7.5a.25.25 0 0 1 .25-.25H3V4z"></path></svg>`;

const WRAP_ICON = `<svg viewBox="0 0 16 16" focusable="false"><path d="M2 3.75A.75.75 0 0 1 2.75 3h8.5a2.75 2.75 0 1 1 0 5.5H7.56l1.22 1.22a.75.75 0 1 1-1.06 1.06L5.22 8.28a.75.75 0 0 1 0-1.06l2.5-2.5a.75.75 0 1 1 1.06 1.06L7.56 7h3.69a1.25 1.25 0 1 0 0-2.5h-8.5A.75.75 0 0 1 2 3.75z"></path><path d="M2 7.75A.75.75 0 0 1 2.75 7h1.5a.75.75 0 0 1 0 1.5h-1.5A.75.75 0 0 1 2 7.75zm0 4A.75.75 0 0 1 2.75 11h6.5a.75.75 0 0 1 0 1.5h-6.5A.75.75 0 0 1 2 11.75z"></path></svg>`;

function languageLabel(lang) {
  return LANGUAGE_LABELS[lang] ?? lang.toUpperCase();
}

function buildCodebox(lang, shikiHtml) {
  const label = languageLabel(lang);
  return `<div class="codebox" data-language="${lang}" data-theme="${THEME}">` +
    `<div class="codebox__toolbar">` +
    `<span class="codebox__language">${label}</span>` +
    `<div class="codebox__actions">` +
    `<button class="codebox__copy" type="button" aria-label="Copy ${label} code to clipboard" data-copy-state="idle">` +
    `<span class="codebox__copy-icon" aria-hidden="true">${COPY_ICON}</span>` +
    `<span class="codebox__copy-label">Copy</span></button>` +
    `<button class="codebox__wrap" type="button" aria-label="Toggle word wrap for ${label} code" aria-pressed="false">` +
    `<span class="codebox__wrap-icon" aria-hidden="true">${WRAP_ICON}</span>` +
    `<span class="codebox__wrap-label">Wrap</span></button>` +
    `</div></div>` +
    shikiHtml +
    `</div>`;
}

function collectHtmlFiles(dir) {
  const results = [];
  for (const entry of readdirSync(dir)) {
    const full = join(dir, entry);
    if (statSync(full).isDirectory()) {
      results.push(...collectHtmlFiles(full));
    } else if (extname(full) === '.html') {
      results.push(full);
    }
  }
  return results;
}

async function upgradeFile(filePath) {
  const src = readFileSync(filePath, 'utf8');
  const $ = load(src, { decodeEntities: false });

  const blocks = $('div.highlighter-rouge').not('.language-mermaid').toArray();
  if (!blocks.length) return 0;

  for (const el of blocks) {
    const classes = (el.attribs.class ?? '').split(/\s+/);
    const langClass = classes.find(c => c.startsWith('language-'));
    const lang = langClass ? langClass.slice('language-'.length) : 'text';

    const code = $(el).find('pre code').text();

    let shikiHtml;
    try {
      shikiHtml = await codeToHtml(code, { lang, theme: THEME });
    } catch {
      shikiHtml = await codeToHtml(code, { lang: 'text', theme: THEME });
    }

    $(el).replaceWith(buildCodebox(lang, shikiHtml));
  }

  writeFileSync(filePath, $.html(), 'utf8');
  return blocks.length;
}

async function main() {
  const siteDir = process.argv[2];
  if (!siteDir) {
    console.error('Usage: node shiki-upgrade.mjs <site-dir>');
    process.exit(1);
  }

  const files = collectHtmlFiles(siteDir);
  let totalBlocks = 0;
  for (const file of files) {
    const count = await upgradeFile(file);
    if (count) console.log(`  ${count} block(s): ${file}`);
    totalBlocks += count;
  }
  console.log(`shiki-upgrade: upgraded ${totalBlocks} block(s) across ${files.length} file(s)`);
}

main().catch(err => {
  console.error(err);
  process.exit(1);
});
