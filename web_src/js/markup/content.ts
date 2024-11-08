import {renderMermaid} from './mermaid';
import {renderMath} from './math';
import {renderCodeCopy} from './codecopy';
import {renderAsciicast} from './asciicast';
import {initMarkupTasklist} from './tasklist';

// code that runs for all markup content
export function initMarkupContent() {
  renderMermaid();
  renderMath();
  renderCodeCopy();
  renderAsciicast();
}

// code that only runs for comments
export function initCommentContent() {
  initMarkupTasklist();
}
