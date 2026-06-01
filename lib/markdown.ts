import { marked } from "marked";
import DOMPurify from "dompurify";

export function renderMarkdown(src: string): string {
  return DOMPurify.sanitize(marked.parse(src) as string);
}
