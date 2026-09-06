"use client";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";

export function safeLink(url: string) {
  try {
    const parsed = new URL(url);
    return ["https:", "http:"].includes(parsed.protocol) ? url : "";
  } catch {
    return url.startsWith("/") && !url.startsWith("//") ? url : "";
  }
}
export function SafeMarkdown({ children }: { children: string }) {
  return (
    <div className="space-y-3 break-words text-sm leading-7 [&_p]:my-2 [&_ul]:list-disc [&_ul]:pl-5 [&_ol]:list-decimal [&_ol]:pl-5 [&_pre]:overflow-auto [&_pre]:rounded-lg [&_pre]:bg-builder-surface [&_pre]:p-3 [&_code]:font-mono [&_code]:text-xs">
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        skipHtml
        urlTransform={safeLink}
        disallowedElements={[
          "img",
          "iframe",
          "script",
          "style",
          "form",
          "input",
        ]}
        components={{
          a: ({ href, children }) =>
            href ? (
              <a
                href={href}
                target="_blank"
                rel="noopener noreferrer"
                className="underline underline-offset-4"
              >
                {children}
              </a>
            ) : (
              <span>{children}</span>
            ),
        }}
      >
        {children}
      </ReactMarkdown>
    </div>
  );
}
