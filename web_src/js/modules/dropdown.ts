// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

// Details can be opened by clicking summary or by pressing Space or Enter while
// being focused on summary. But without JS options for closing it are limited.
// Event listeners in this file provide more convenient options for that:
// click iteration with anything on the page and pressing Escape.

// FixMe: HTMX makes this ineffective!
function markDropdowns() {
  const dropdowns = document.querySelectorAll<HTMLDetailsElement>('details.dropdown');
  for (const dropdown of dropdowns) {
    dropdown.classList.add('js-enhanced');
  }
}

export function initDropdowns() {
  document.addEventListener('click', (event) => {
    const dropdown = document.querySelector<HTMLDetailsElement>('details.dropdown[open]');
    // No open dropdowns on page, nothing to do.
    if (dropdown === null) return;

    const target = event.target as HTMLElement;
    // User clicked something in the open dropdown, don't interfere.
    if (dropdown.contains(target)) return;

    // User clicked something that isn't the open dropdown, so close it.
    dropdown.removeAttribute('open');
  });

  // Close open dropdowns on Escape press
  document.addEventListener('keydown', (event) => {
    // This press wasn't escape, nothing to do.
    if (event.key !== 'Escape') return;

    const dropdown = document.querySelector<HTMLDetailsElement>('details.dropdown[open]');
    // No open dropdowns on page, nothing to do.
    if (dropdown === null) return;

    // User pressed Escape while having an open dropdown, probably wants it be closed.
    dropdown.removeAttribute('open');
  });

  markDropdowns();
}
