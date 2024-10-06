<script>
import {SvgIcon} from '../svg.js';
import {contrastColor} from '../utils/color.js';
import {GET} from '../modules/fetch.js';
import {emojiHTML} from '../features/emoji.js';
import {htmlEscape} from 'escape-goat';

const {appSubUrl, i18n} = window.config;

export default {
  components: {SvgIcon},
  data: () => ({
    loading: false,
    issue: null,
    i18nErrorOccurred: i18n.error_occurred,
    i18nErrorMessage: null,
  }),
  computed: {
    createdAt() {
      return new Date(this.issue.created_at).toLocaleDateString(undefined, {year: 'numeric', month: 'short', day: 'numeric'});
    },

    body() {
      const body = this.issue.body.replace(/\n+/g, ' ');
      if (body.length > 85) {
        return `${body.substring(0, 85)}…`;
      }
      return body;
    },

    icon() {
      if (this.issue.pull_request !== null) {
        if (this.issue.pull_request.merged === true) {
          return 'octicon-git-merge'; // Merged PR
        }

        if (this.issue.state === 'closed') {
          return 'octicon-git-pull-request-closed'; // Closed PR
        }

        if (this.issue.pull_request.draft === true) {
          return 'octicon-git-pull-request-draft'; // WIP PR
        }

        return 'octicon-git-pull-request'; // Open PR
      }

      if (this.issue.state === 'closed') {
        return 'octicon-issue-closed'; // Closed issue
      }

      return 'octicon-issue-opened'; // Open issue
    },

    color() {
      if (this.issue.pull_request !== null) {
        if (this.issue.pull_request.merged === true) {
          return 'purple'; // Merged PR
        }

        if (this.issue.pull_request.draft === true && this.issue.state === 'open') {
          return 'grey'; // WIP PR
        }
      }

      if (this.issue.state === 'closed') {
        return 'red'; // Closed issue
      }

      return 'green'; // Open issue
    },

    labels() {
      return this.issue.labels.map((label) => ({
        name: htmlEscape(label.name).replaceAll(/:[-+\w]+:/g, (emoji) => {
          return emojiHTML(emoji.substring(1, emoji.length - 1));
        }),
        color: `#${label.color}`,
        textColor: contrastColor(`#${label.color}`),
      }));
    },
  },
  mounted() {
    this.$refs.root.addEventListener('ce-load-context-popup', (e) => {
      const data = e.detail;
      if (!this.loading && this.issue === null) {
        this.load(data);
      }
    });
  },
  methods: {
    async load(data) {
      this.loading = true;
      this.i18nErrorMessage = null;

      try {
        const response = await GET(`${appSubUrl}/${data.owner}/${data.repo}/issues/${data.index}/info`);
        const respJson = await response.json();
        if (!response.ok) {
          this.i18nErrorMessage = respJson.message ?? i18n.network_error;
          return;
        }
        this.issue = respJson;
      } catch {
        this.i18nErrorMessage = i18n.network_error;
      } finally {
        this.loading = false;
      }
    },
  },
};
</script>
<template>
  <div ref="root">
    <div v-if="loading" class="tw-h-12 tw-w-12 is-loading"/>
    <div v-if="!loading && issue !== null" id="issue-info-popup">
      <p><small>{{ issue.repository.full_name }} on {{ createdAt }}</small></p>
      <p><svg-icon :name="icon" :class="['text', color]"/> <strong>{{ issue.title }}</strong> #{{ issue.number }}</p>
      <p>{{ body }}</p>
      <div class="labels-list">
        <!-- eslint-disable-next-line vue/no-v-html -->
        <div v-for="label in labels" :key="label.name" class="ui label" :style="{ color: label.textColor, backgroundColor: label.color }" v-html="label.name"/>
      </div>
    </div>
    <div v-if="!loading && issue === null">
      <p><small>{{ i18nErrorOccurred }}</small></p>
      <p>{{ i18nErrorMessage }}</p>
    </div>
  </div>
</template>
