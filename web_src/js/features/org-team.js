import $ from 'jquery';

const {appSubUrl} = window.config;

export function initOrgTeamSearchRepoBox() {
  const $searchRepoBox = $('#search-repo-box');
  $searchRepoBox.search({
    minCharacters: 2,
    apiSettings: {
      url: `${appSubUrl}/repo/search?q={query}&uid=${$searchRepoBox.data('uid')}`,
      onResponse(response) {
        const items = [];
        for (const item of response.data) {
          items.push({
            title: item.repository.full_name.split('/')[1],
            description: item.repository.full_name,
          });
        }
        return {results: items};
      },
    },
    searchFields: ['full_name'],
    showNoResults: false,
  });
}
