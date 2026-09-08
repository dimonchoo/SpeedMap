#!/usr/bin/env node
const fs = require('fs');
const path = require('path');

const ROOT = path.resolve(__dirname, '..');
const TEMPLATE_PATH = path.join(ROOT, 'frontend/index.template.html');
const OUTPUT_PATH = path.join(ROOT, 'frontend/index.html');

const startTime = Date.now();

if (!fs.existsSync(TEMPLATE_PATH)) {
  console.error('❌ Error: Template not found at', TEMPLATE_PATH);
  process.exit(1);
}

let template = fs.readFileSync(TEMPLATE_PATH, 'utf8');

const includeRegex = /<!--\s*include:\s*([\w.\-\/]+)\s*-->/g;
const includedFiles = [];

template = template.replace(includeRegex, (fullMatch, relPath) => {
  const fullIncludePath = path.join(ROOT, 'frontend', relPath);
  if (!fs.existsSync(fullIncludePath)) {
    console.error('❌ Error: Include file not found:', fullIncludePath);
    process.exit(1);
  }
  const content = fs.readFileSync(fullIncludePath, 'utf8');
  includedFiles.push(relPath);
  return content;
});

// HTML & Template Tag Validation
// Void elements that do not require closing tags
const voidElements = new Set([
  'area', 'base', 'br', 'col', 'embed', 'hr', 'img', 'input', 
  'link', 'meta', 'param', 'source', 'track', 'wbr', '!doctype'
]);

// Tags whose contents shouldn't be parsed as normal HTML tags (e.g. scripts, styles)
const rawTextElements = new Set(['script', 'style']);

// Tags that don't need strict closing or can be self-contained in SVG/HTML
const ignoreBalance = new Set(['path', 'circle', 'rect', 'line', 'polygon', 'polyline', 'ellipse', 'use', 'stop']);

const stack = [];
// Match tags: <tag, </tag, with attributes
const tagRegex = /<\/?([a-zA-Z0-9_\-:]+)([^>]*?)(\/?)>/g;
let tagMatch;
let tagErrors = 0;

let inRawElement = null;

while ((tagMatch = tagRegex.exec(template)) !== null) {
  const fullTag = tagMatch[0];
  const tagName = tagMatch[1].toLowerCase();
  const isClosing = fullTag.startsWith('</');
  const isSelfClosing = tagMatch[3] === '/' || voidElements.has(tagName) || ignoreBalance.has(tagName);

  if (inRawElement) {
    if (isClosing && tagName === inRawElement) {
      inRawElement = null;
    }
    continue;
  }

  if (rawTextElements.has(tagName) && !isClosing) {
    inRawElement = tagName;
    continue;
  }

  if (isSelfClosing) continue;

  if (isClosing) {
    if (stack.length === 0) {
      console.error('❌ Unexpected closing tag: </' + tagName + '> at index ' + tagMatch.index);
      tagErrors++;
    } else {
      const top = stack[stack.length - 1];
      if (top.tagName === tagName) {
        stack.pop();
      } else {
        const line = template.slice(0, tagMatch.index).split('\n').length;
        console.error('❌ Tag mismatch: found </' + tagName + '> at line ' + line + ', expected </' + top.tagName + '> (opened at line ' + top.line + ')');
        tagErrors++;
        stack.pop();
      }
    }
  } else {
    const line = template.slice(0, tagMatch.index).split('\n').length;
    stack.push({ tagName, line });
  }
}

if (stack.length > 0) {
  console.error('❌ Unclosed tags remaining (' + stack.length + '):', stack.slice(-5));
  tagErrors += stack.length;
}

if (tagErrors > 0) {
  console.error('🚨 Build failed: ' + tagErrors + ' HTML tag errors detected.');
  process.exit(1);
}

fs.writeFileSync(OUTPUT_PATH, template, 'utf8');
const elapsed = Date.now() - startTime;
const outputLines = template.split('\n').length;

console.log('✅ HTML compiled successfully in ' + elapsed + 'ms:');
console.log('   📦 ' + includedFiles.length + ' partials assembled');
console.log('   📄 ' + outputLines + ' lines generated -> ' + path.relative(ROOT, OUTPUT_PATH));
console.log('   🛡️  Tag validation: 0 errors (100% clean balanced DOM)');
