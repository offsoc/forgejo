<script>
import {Bar} from 'vue-chartjs';
import {
  Chart,
  Tooltip,
  BarElement,
  CategoryScale,
  LinearScale,
} from 'chart.js';
import {chartJsColors} from '../utils/color.js';
import {createApp} from 'vue';

Chart.defaults.color = chartJsColors.text;
Chart.defaults.borderColor = chartJsColors.border;

Chart.register(
  CategoryScale,
  LinearScale,
  BarElement,
  Tooltip,
);

const sfc = {
  components: {Bar},
  props: {
    locale: {
      type: Object,
      required: true,
    },
  },
  data: () => ({
    colors: {
      barColor: 'green',
    },

    // possible keys:
    // * avatar_link: (...)
    // * commits: (...)
    // * home_link: (...)
    // * login: (...)
    // * name: (...)
    activityTopAuthors: window.config.pageData.repoActivityTopAuthors || [],
    i18nCommitActivity: this,
  }),
  methods: {
    graphPoints() {
      return {
        datasets: [{
          label: this.locale.commitActivity,
          data: this.activityTopAuthors.map((item) => item.commits),
          backgroundColor: this.colors.barColor,
          barThickness: 40,
          borderWidth: 0,
          tension: 0.3,
        }],
        labels: this.activityTopAuthors.map((item) => item.name),
      };
    },
    getOptions() {
      return {
        responsive: true,
        maintainAspectRatio: false,
        animation: true,
        scales: {
          x: {
            type: 'category',
            grid: {
              display: false,
            },
            ticks: {
              color: 'transparent', // Disable drawing of labels on the x-axis.
            },
          },
          y: {
            ticks: {
              stepSize: 1,
            },
          },
        },
      };
    },
  },
  mounted() {
    const refStyle = window.getComputedStyle(this.$refs.style);
    this.colors.barColor = refStyle.backgroundColor;

    for (const item of this.activityTopAuthors) {
      const img = new Image();
      img.src = item.avatar_link;
      item.avatar_img = img;
    }

    Chart.register({
      id: 'image_label',
      afterDraw: (chart) => {
        const xAxis = chart.boxes[0];
        const yAxis = chart.boxes[1];
        for (const [index] of xAxis.ticks.entries()) {
          const x = xAxis.getPixelForTick(index);
          const img = this.activityTopAuthors[index].avatar_img;

          chart.ctx.save();
          chart.ctx.drawImage(img, 0, 0, img.naturalWidth, img.naturalHeight, x - 10, yAxis.bottom + 10, 20, 20);
          chart.ctx.restore();
        }
      },
      beforeEvent: (chart, args) => {
        const event = args.event;
        if (event.type !== 'mousemove' && event.type !== 'click') return;

        const yAxis = chart.boxes[1];
        if (event.y < yAxis.bottom + 10 || event.y > yAxis.bottom + 30) {
          chart.canvas.style.cursor = '';
          return;
        }

        const xAxis = chart.boxes[0];
        const pointIdx = xAxis.ticks.findIndex((_, index) => {
          const x = xAxis.getPixelForTick(index);
          return event.x >= x - 10 && event.x <= x + 10;
        });

        if (pointIdx === -1) {
          chart.canvas.style.cursor = '';
          return;
        }

        chart.canvas.style.cursor = 'pointer';
        if (event.type === 'click' && this.activityTopAuthors[pointIdx].home_link) {
          window.location.href = this.activityTopAuthors[pointIdx].home_link;
        }
      },
    });
  },
};

export function initRepoActivityTopAuthorsChart() {
  const el = document.getElementById('repo-activity-top-authors-chart');
  if (el) {
    createApp(sfc, {
      locale: {
        commitActivity: el.getAttribute('data-locale-commit-activity'),
      },
    }).mount(el);
  }
}

export default sfc; // activate the IDE's Vue plugin
</script>
<template>
  <div>
    <div class="activity-bar-graph" ref="style" style="width: 0; height: 0;"/>
    <Bar height="150px" :data="graphPoints()" :options="getOptions()"/>
  </div>
</template>
