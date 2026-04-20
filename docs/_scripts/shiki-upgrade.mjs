import { codeToHtml } from 'shiki';
import { load } from 'cheerio';
import { readFileSync, writeFileSync, readdirSync, statSync } from 'fs';
import { join, extname } from 'path';

const THEME = 'github-dark';

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

    let html;
    try {
      html = await codeToHtml(code, { lang, theme: THEME });
    } catch {
      html = await codeToHtml(code, { lang: 'text', theme: THEME });
    }

    $(el).replaceWith(html);
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
