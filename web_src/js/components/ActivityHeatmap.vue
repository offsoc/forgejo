<script lang="ts">
import {CalendarHeatmap} from 'vue3-calendar-heatmap';

export default {
  components: {CalendarHeatmap},
  props: {
    values: {
      type: Array,
      default: () => [],
    },
    locale: {
      type: Object,
      default: () => {},
    },
  },
  data: () => ({
    colorRange: [
      'var(--color-secondary-alpha-60)',
      'var(--color-secondary-alpha-60)',
      'var(--color-primary-light-4)',
      'var(--color-primary-light-2)',
      'var(--color-primary)',
      'var(--color-primary-dark-2)',
      'var(--color-primary-dark-4)',
    ],
    endDate: new Date(),
  }),
  mounted() {
    // work around issue with first legend color being rendered twice and legend cut off
    const legend = document.querySelector('.vch__external-legend-wrapper');
    legend.setAttribute('viewBox', '12 0 80 10');
    legend.style.marginRight = '-12px';
  },
  methods: {
    handleDayClick(e) {
      // Reset filter if same date is clicked
      const params = new URLSearchParams(document.location.search);
      const queryDate = params.get('date');
      // Timezone has to be stripped because toISOString() converts to UTC
      const clickedDate = new Date(e.date - (e.date.getTimezoneOffset() * 60000)).toISOString().substring(0, 10);

      if (queryDate && queryDate === clickedDate) {
        params.delete('date');
      } else {
        params.set('date', clickedDate);
      }

      params.delete('page');

      const newSearch = params.toString();
      window.location.search = newSearch.length ? `?${newSearch}` : '';
    },
  },
};
</script>
<template>
  <div class="total-contributions">
    {{ locale.contributions_in_the_last_12_months }}
  </div>
  <calendar-heatmap
    :locale="locale"
    :no-data-text="locale.contributions_zero"
    :tooltip-formatter="
      (v) =>
        locale.contributions_format
          .replace(
            '{contributions}',
            `<b>${v.count} ${
              v.count === 1
                ? locale.contributions_one
                : locale.contributions_few
            }</b>`
          )
          .replace('{month}', locale.months[v.date.getMonth()])
          .replace('{day}', v.date.getDate())
          .replace('{year}', v.date.getFullYear())
    "
    :end-date="endDate"
    :values="values"
    :range-color="colorRange"
    @day-click="handleDayClick($event)"
  />
</template>
