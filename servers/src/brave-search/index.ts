import { list } from './list.js';
import { call } from './tools.js';

const infos = async () => {
	return {
		name: 'brave-search',
		displayName: 'Brave Search',
		categories: ['search'],
		integration: 'brave-search',
		description: 'Search the web using Brave\'s search engine',
		icon: 'https://cdn.search.brave.com/serp/v2/_app/immutable/assets/brave-logo-small.1fMdoHsa.svg',
		url: 'https://api-dashboard.search.brave.com/app/keys',
		form: {
			config: {},
			secrets: {
				braveApiKey: {
					description: 'API Key',
					label: 'API Key',
					required: true,
				},
			},
		},
	};
};

export { call, infos, list };
