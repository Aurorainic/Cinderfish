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
    const trimmed = url.replace(/[.,;:!?)]+$/, "");
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

  return text.replace(new RegExp(`${PLACEHOLDER}(\\d+)${PLACEHOLDER}`, "g"), (_m, index: string) => {
    return stashed[Number(index)] ?? "";
  });
}

export function isImageType(type: string): boolean {
  return type.startsWith("image/");
}
