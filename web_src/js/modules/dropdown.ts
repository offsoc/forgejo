// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

// Details can be opened by clicking summary or by pressing Space or Enter while
// being focused on summary. But without JS options for closing it are limited.
// Event listeners in this file provide more convenient options for that:
// click iteration with anything on the page and pressing Escape.

function markDropdowns() {
  const dropdowns = document.querySelectorAll<HTMLDetailsElement>('details.dropdown')
  dropdowns.forEach((dropdown) => {
    dropdown.classList.add("js-enhanced");
  });
}

export function initDropdowns() {
  document.addEventListener('click', (event) => {
    const dropdown = document.querySelector<HTMLDetailsElement>('details.dropdown[open]');
    if (dropdown == null)
      // No open dropdowns on page, nothing to do.
      return;

    const target = event.target as HTMLElement;
    if (dropdown.contains(target))
      // User clicked something in the open dropdown, don't interfere.
      return;

    // User clicked something that isn't the open dropdown, so close it.
    dropdown.removeAttribute('open');
  });

  // Close open dropdowns on Escape press
  document.addEventListener('keydown', (event) => {
    if (event.key !== 'Escape')
      // This press wasn't escape, nothing to do.
      return;

    const dropdown = document.querySelector<HTMLDetailsElement>('details.dropdown[open]');
    if (dropdown == null)
      // No open dropdowns on page, nothing to do.
      return;

    // User pressed Escape while having an open dropdown, probably wants it be closed.
    dropdown.removeAttribute('open');
  });

  markDropdowns();
}
