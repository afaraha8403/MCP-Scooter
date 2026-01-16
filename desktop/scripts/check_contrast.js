import fs from 'fs';
import path from 'path';

// Helper: convert hex color to RGB array
function hexToRgb(hex) {
    let clean = hex.trim().replace('#', '');
    if (clean.length === 3) {
        clean = clean.split('').map(c => c + c).join('');
    }
    const num = parseInt(clean, 16);
    return [(num >> 16) & 255, (num >> 8) & 255, num & 255];
}

// Helper: calculate relative luminance
function luminance(rgb) {
    const srgb = rgb.map(v => {
        const c = v / 255;
        return c <= 0.03928 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4);
    });
    return 0.2126 * srgb[0] + 0.7152 * srgb[1] + 0.0722 * srgb[2];
}

function contrastRatio(fgHex, bgHex) {
    const fg = hexToRgb(fgHex);
    const bg = hexToRgb(bgHex);
    const L1 = luminance(fg);
    const L2 = luminance(bg);
    const lighter = Math.max(L1, L2);
    const darker = Math.min(L1, L2);
    return (lighter + 0.05) / (darker + 0.05);
}

// Resolve __dirname in ES module
const __dirname = path.dirname(new URL(import.meta.url).pathname);

// Read CSS file
const cssPath = path.resolve(__dirname, '..', 'src', 'App.css');
const css = fs.readFileSync(cssPath, 'utf8');

function extractVars(sectionRegex) {
    const match = css.match(sectionRegex);
    if (!match) return {};
    const body = match[1];
    const varMap = {};
    const varLines = body.split(/\n/);
    varLines.forEach(line => {
        const varMatch = line.match(/(--[\w-]+)\s*:\s*([^;]+);/);
        if (varMatch) {
            varMap[varMatch[1].trim()] = varMatch[2].trim();
        }
    });
    return varMap;
}

const lightVars = extractVars(/:root\s*{([^}]*)}/s);
const darkVars = extractVars(/\[data-theme="dark"\]\s*{([^}]*)}/s);

// Define pairs to test (foreground, background)
const pairs = [
    { name: 'text-primary vs background-base', fg: '--text-primary', bg: '--background-base' },
    { name: 'text-secondary vs background-card', fg: '--text-secondary', bg: '--background-card' },
    { name: 'toolbar-text vs toolbar-bg', fg: '--toolbar-text', bg: '--toolbar-bg' },
    { name: 'accent-primary vs background-base', fg: '--accent-primary', bg: '--background-base' },
];

function evaluate(vars) {
    const results = [];
    let failCount = 0;
    pairs.forEach(p => {
        const fg = vars[p.fg];
        const bg = vars[p.bg];
        if (!fg || !bg) return;
        const ratio = contrastRatio(fg, bg);
        const pass = ratio >= 4.5;
        if (!pass) failCount++;
        results.push({ pair: p.name, foreground: fg, background: bg, ratio: Number(ratio.toFixed(2)), pass });
    });
    return { results, failCount };
}

const lightEval = evaluate(lightVars);
const darkEval = evaluate(darkVars);

const report = {
    light: lightEval.results,
    dark: darkEval.results,
};

fs.writeFileSync(path.resolve(__dirname, 'contrast-report.json'), JSON.stringify(report, null, 2), 'utf8');

if (lightEval.failCount > 0 || darkEval.failCount > 0) {
    console.error('Contrast check failed for some pairs. See contrast-report.json');
    process.exit(1);
} else {
    console.log('All contrast ratios meet WCAG AA requirements.');
    process.exit(0);
}



// Helper: convert hex color to RGB array
function hexToRgb(hex) {
    let clean = hex.trim().replace('#', '');
    if (clean.length === 3) {
        clean = clean.split('').map(c => c + c).join('');
    }
    const num = parseInt(clean, 16);
    return [(num >> 16) & 255, (num >> 8) & 255, num & 255];
}

// Helper: calculate relative luminance
function luminance(rgb) {
    const srgb = rgb.map(v => {
        const c = v / 255;
        return c <= 0.03928 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4);
    });
    return 0.2126 * srgb[0] + 0.7152 * srgb[1] + 0.0722 * srgb[2];
}

function contrastRatio(fgHex, bgHex) {
    const fg = hexToRgb(fgHex);
    const bg = hexToRgb(bgHex);
    const L1 = luminance(fg);
    const L2 = luminance(bg);
    const lighter = Math.max(L1, L2);
    const darker = Math.min(L1, L2);
    return (lighter + 0.05) / (darker + 0.05);
}

// Read CSS file
const cssPath = path.resolve(__dirname, '..', 'src', 'App.css');
const css = fs.readFileSync(cssPath, 'utf8');

function extractVars(sectionRegex) {
    const match = css.match(sectionRegex);
    if (!match) return {};
    const body = match[1];
    const varMap = {};
    const varLines = body.split(/\n/);
    varLines.forEach(line => {
        const varMatch = line.match(/(--[\w-]+)\s*:\s*([^;]+);/);
        if (varMatch) {
            varMap[varMatch[1].trim()] = varMatch[2].trim();
        }
    });
    return varMap;
}

const lightVars = extractVars(/:root\s*{([^}]*)}/s);
const darkVars = extractVars(/\[data-theme="dark"\]\s*{([^}]*)}/s);

// Define pairs to test (foreground, background)
const pairs = [
    { name: 'text-primary vs background-base', fg: '--text-primary', bg: '--background-base' },
    { name: 'text-secondary vs background-card', fg: '--text-secondary', bg: '--background-card' },
    { name: 'toolbar-text vs toolbar-bg', fg: '--toolbar-text', bg: '--toolbar-bg' },
    { name: 'accent-primary vs background-base', fg: '--accent-primary', bg: '--background-base' },
];

function evaluate(vars) {
    const results = [];
    let failCount = 0;
    pairs.forEach(p => {
        const fg = vars[p.fg];
        const bg = vars[p.bg];
        if (!fg || !bg) return; // skip if missing
        const ratio = contrastRatio(fg, bg);
        const pass = ratio >= 4.5;
        if (!pass) failCount++;
        results.push({ pair: p.name, foreground: fg, background: bg, ratio: Number(ratio.toFixed(2)), pass });
    });
    return { results, failCount };
}

const lightEval = evaluate(lightVars);
const darkEval = evaluate(darkVars);

const report = {
    light: lightEval.results,
    dark: darkEval.results,
};

fs.writeFileSync(path.resolve(__dirname, 'contrast-report.json'), JSON.stringify(report, null, 2), 'utf8');

if (lightEval.failCount > 0 || darkEval.failCount > 0) {
    console.error('Contrast check failed for some pairs. See contrast-report.json');
    process.exit(1);
} else {
    console.log('All contrast ratios meet WCAG AA requirements.');
    process.exit(0);
}
