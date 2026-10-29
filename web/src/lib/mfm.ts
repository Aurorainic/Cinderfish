// A small MFM renderer for note bodies. The backend stores MFM text, not
// HTML, so the frontend is responsible for rendering it. Everything is
// escaped before any markup is added; only the constructs below produce
// HTML, so user text can never inject tags or attributes.

const PLACEHOLDER = "\u0000";

function escapeHtml(text: string): string {
  return text
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");
}

export function renderMFM(input: string | null | undefined): string {
  if (!input) return "";
  const stashed: string[] = [];
  const stash = (html: string): string => {
    stashed.push(html);
    return `${PLACEHOLDER}${stashed.length - 1}${PLACEHOLDER}`;
  };

  // Fenced code first, so URLs and markup inside it stay literal.
  let text = input.replace(
    /```([a-zA-Z0-9+-]*)\n?([\s\S]*?)```/g,
    (_m, _lang: string, code: string) =>
      stash(`<pre class="mfm-code"><code>${escapeHtml(code.replace(/\n$/, ""))}</code></pre>`),
  );
  text = text.replace(/`([^`\n]+)`/g, (_m, code: string) =>
    stash(`<code class="mfm-inline-code">${escapeHtml(code)}</code>`),
  );

  text = escapeHtml(text);

  text = text.replace(/https?:\/\/[^\s<]+/g, (url) => {
    const trimmed = url.replace(/[.,;:!?)\]]+$/, "");
    const tail = url.slice(trimmed.length);
    return `<a class="mfm-link" href="${trimmed}" target="_blank" rel="noopener noreferrer">${trimmed}</a>${tail}`;
  });

  text = text.replace(
    /(^|[\s(])@([A-Za-z0-9_]+)(?:@([A-Za-z0-9.-]+))?/g,
    (_m, lead: string, name: string, host?: string) => {
      const handle = host ? `@${name}@${host}` : `@${name}`;
      return `${lead}<a class="mfm-mention" href="/${handle}">${handle}</a>`;
    },
  );

  text = text.replace(/(^|[\s(])#([^\s#<>]+)/g, (_m, lead: string, tag: string) => {
    return `${lead}<a class="mfm-hashtag" href="/explore?tag=${encodeURIComponent(tag)}">#${tag}</a>`;
  });

  // Custom emoji are shown inline; without an emoji dictionary they keep
  // their :name: form but stop being mistaken for text.
  text = text.replace(/:([A-Za-z0-9_+-]+):/g, '<span class="mfm-emoji">:$1:</span>');

  text = text
    .replace(/\*\*([^*\n]+)\*\*/g, "<strong>$1</strong>")
    .replace(/~~([^~\n]+)~~/g, "<del>$1</del>")
    .replace(/(^|[^*])\*([^*\n]+)\*/g, "$1<em>$2</em>");

  text = text.replace(/\n/g, "<br />");

  text = renderFunctions(text, 0);

  return text.replace(new RegExp(`${PLACEHOLDER}(\\d+)${PLACEHOLDER}`, "g"), (_m, index: string) => {
    return stashed[Number(index)] ?? "";
  });
}

// MFM function support ($[x2 text], $[spin.left text], ...). The pass runs
// after every other transform, so the content inside a function is already
// escaped HTML and the function layer only wraps that HTML in elements with
// attributes it produces itself; user text can never inject markup here.
// Unknown or invalid functions degrade to their (already rendered) content.
const MAX_MFM_DEPTH = 40;
const FUNCTION_TOKEN = /^[a-zA-Z0-9_.=,+-]+$/;
const HEX_COLOR = /^[0-9a-fA-F]{3,8}$/;

function findFunctionClose(text: string, start: number): number {
  // The opening "$[" before `start` is already consumed, so begin one level deep.
  let depth = 1;
  for (let i = start; i < text.length; i++) {
    if (text.startsWith("$[", i)) {
      depth++;
      i++;
      continue;
    }
    if (text[i] === "]") {
      depth--;
      if (depth === 0) return i;
    }
  }
  return -1;
}

function renderFunctions(text: string, depth: number): string {
  if (depth > MAX_MFM_DEPTH || !text.includes("$[")) return text;
  let out = "";
  let i = 0;
  for (;;) {
    const open = text.indexOf("$[", i);
    if (open === -1) {
      out += text.slice(i);
      break;
    }
    out += text.slice(i, open);
    const close = findFunctionClose(text, open + 2);
    if (close === -1) {
      // Unclosed function: keep the literal text and keep scanning.
      out += "$[";
      i = open + 2;
      continue;
    }
    out += renderFunctionCall(text.slice(open + 2, close), depth);
    i = close + 1;
  }
  return out;
}

function renderFunctionCall(inner: string, depth: number): string {
  const sep = inner.search(/\s/);
  const token = sep === -1 ? inner : inner.slice(0, sep);
  const content = sep === -1 ? "" : inner.slice(sep + 1);
  const processed = renderFunctions(content, depth + 1);

  // Misskey argument syntax: the first token is `name.arg1,arg2` where each
  // arg may be a flag or key=value. Malformed tokens degrade to the content.
  if (!FUNCTION_TOKEN.test(token)) return processed;
  const dot = token.indexOf(".");
  const name = dot === -1 ? token : token.slice(0, dot);
  const argText = dot === -1 ? "" : token.slice(dot + 1);
  const flags: string[] = [];
  const params: Record<string, string> = {};
  for (const part of argText.split(",")) {
    if (!part) continue;
    const eq = part.indexOf("=");
    if (eq === -1) flags.push(part);
    else params[part.slice(0, eq)] = part.slice(eq + 1);
  }

  switch (name) {
    case "x2":
    case "x3":
    case "x4":
      return `<span class="mfm-${name}">${processed}</span>`;
    case "spin": {
      const axis = flags.includes("x")
        ? " mfm-spin--x"
        : flags.includes("y")
          ? " mfm-spin--y"
          : "";
      const dir = [
        flags.includes("left") ? "reverse" : "",
        flags.includes("alternate") || flags.includes("alt") ? "alternate" : "",
      ]
        .filter(Boolean)
        .join(" ");
      const speed = sanitizeDuration(params.speed ?? "");
      const style = [dir ? `animation-direction:${dir}` : "", speed ? `animation-duration:${speed}` : ""]
        .filter(Boolean)
        .join(";");
      return `<span class="mfm-spin${axis}"${style ? ` style="${style}"` : ""}>${processed}</span>`;
    }
    case "fg":
    case "bg": {
      const color = params.color;
      if (color && HEX_COLOR.test(color)) {
        const prop = name === "fg" ? "color" : "background-color";
        return `<span style="${prop}:#${color.toLowerCase()}">${processed}</span>`;
      }
      return processed;
    }
    case "font": {
      if (flags.includes("monospace")) return `<span class="mfm-mono">${processed}</span>`;
      if (flags.includes("serif")) return `<span class="mfm-serif">${processed}</span>`;
      return processed;
    }
    case "shake":
      return `<span class="mfm-shake">${processed}</span>`;
    case "jump":
      return `<span class="mfm-jump">${processed}</span>`;
    case "blink":
      return `<span class="mfm-blink">${processed}</span>`;
    case "rainbow":
      return `<span class="mfm-rainbow">${processed}</span>`;
    case "blur":
      return `<span class="mfm-blur">${processed}</span>`;
    case "center":
      return `<span class="mfm-center">${processed}</span>`;
    case "small":
      return `<span class="mfm-small">${processed}</span>`;
    case "plain":
      return `<span class="mfm-plain">${processed}</span>`;
    default:
      return processed;
  }
}

function sanitizeDuration(value: string): string {
  if (/^\d*\.?\d+(ms|s)$/.test(value)) return value;
  if (/^\d*\.?\d+$/.test(value)) return `${value}s`;
  return "";
}

export function isImageType(type: string): boolean {
  return type.startsWith("image/");
}
