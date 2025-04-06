import $ from 'jquery';

class Dimmer {
  dimmerEl: HTMLDivElement;
  active: boolean;

  constructor() {
    this.dimmerEl = document.querySelector('body > div.ui.dimmer') as HTMLDivElement;
    if (!this.dimmerEl) {
      this.dimmerEl = document.createElement('div');
      this.dimmerEl.classList.add('ui', 'dimmer', 'transition');
      document.body.append(this.dimmerEl);
    }
  }

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  dimmer(functionName: string, ...args: any[]) {
    if (functionName === 'add content') {
      this.dimmerEl.append(args[0][0]);
    } else if (functionName === 'get dimmer') {
      return $(this.dimmerEl);
    } else if (functionName === 'show') {
      this.dimmerEl.classList.add('active');
      this.dimmerEl.classList.remove('hidden');
      this.active = true;
    } else if (functionName === 'hide') {
      this.dimmerEl.classList.remove('active');
      this.dimmerEl.classList.add('hidden');
      this.active = false;
    } else if (functionName === 'is active') {
      return this.active;
    }
  }

  // JQuery compatibility functions.
  get(_index: number): HTMLElement {
    return document.body;
  }
  removeClass() {}
  hasClass() {}
}

export function initDimmer() {
  $.fn.dimmer = (arg: string | object) => {
    if (typeof arg === 'object') return new Dimmer();
  };
}
