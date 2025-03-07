import { describe, expect, it } from 'vitest';
import { call, list } from '../src/google-maps';

import { Call, DefineSecret, StandardDecode } from './blaxel';

describe('GoogleMap', async () => {
	// @ts-ignore
	const apiKey = import.meta.env.GOOGLE_MAPS_API_KEY;
	DefineSecret('apikey', apiKey);

	it('listTools', async () => {
		const { tools } = await list();
		expect(tools).toBeDefined();
		expect(tools.length).toBeGreaterThan(0);
		for (let tool of tools) {
			expect(tool.name).toBeDefined();
			expect(tool.description).toBeDefined();
			expect(tool.inputSchema).toBeDefined();
		}
		let listFunc = [
			'maps_geocode',
			'maps_reverse_geocode',
			'maps_search_places',
			'maps_place_details',
			'maps_distance_matrix',
			'maps_elevation',
			'maps_directions',
		];
		// asert it's inside the list
		expect(listFunc.every((func) => tools.some((tool) => tool.name === func))).toBe(true);
	});

	it('maps_geocode', async () => {
		let response = await Call(call, 'maps_geocode', {
			address: 'Gare de troyes',
		});
		let value = StandardDecode(response);
		expect(typeof value.location.lat).toBe('number');
		expect(typeof value.location.lng).toBe('number');
		expect(typeof value.formatted_address).toBe('string');
	});

	it('maps_reverse_geocode', async () => {
		let response = await Call(call, 'maps_reverse_geocode', {
			latitude: 48.2960451,
			longitude: 4.0651728,
		});
		let value = StandardDecode(response);
		expect(typeof value.formatted_address).toBe('string');
		expect(typeof value.place_id).toBe('string');
		expect(value.address_components).toBeDefined();
		expect(value.address_components.length).toBeGreaterThan(0);
		expect(typeof value.address_components[0].long_name).toBe('string');
		expect(typeof value.address_components[0].short_name).toBe('string');
		expect(value.address_components[0].types.length).toBeGreaterThan(0);
	});

	it('maps_search_places', async () => {
		let response = await Call(call, 'maps_search_places', {
			query: 'restaurant',
			location: {
				latitude: 48.2960451,
				longitude: 4.0651728,
			},
		});
		let value = StandardDecode(response);
		expect(value.places.length).toBeGreaterThan(0);
		expect(typeof value.places[0].name).toBe('string');
		expect(typeof value.places[0].formatted_address).toBe('string');
		expect(typeof value.places[0].place_id).toBe('string');
		expect(typeof value.places[0].location.lat).toBe('number');
		expect(typeof value.places[0].location.lng).toBe('number');
		expect(typeof value.places[0].rating).toBe('number');
		expect(value.places[0].types.length).toBeGreaterThan(0);
	});

	it('maps_place_details', async () => {
		let response = await Call(call, 'maps_place_details', {
			place_id: 'ChIJZ5riqoKZ7kcRZMSDX4mGLHI',
		});
		let value = StandardDecode(response);
		expect(typeof value.name).toBe('string');
		expect(typeof value.formatted_address).toBe('string');
		expect(typeof value.formatted_phone_number).toBe('string');
		expect(typeof value.website).toBe('string');
		expect(typeof value.opening_hours.open_now).toBe('boolean');
	});

	it('maps_distance_matrix', async () => {
		let response = await Call(call, 'maps_distance_matrix', {
			origins: ['Troyes'],
			destinations: ['Paris'],
		});
		let value = StandardDecode(response);
		expect(value.origin_addresses.length).toBeGreaterThan(0);
		expect(value.destination_addresses.length).toBeGreaterThan(0);
		expect(value.results.length).toBeGreaterThan(0);
		expect(value.results[0].elements.length).toBeGreaterThan(0);
		expect(value.results[0].elements.length).toBeGreaterThan(0);
		const res = value.results[0].elements[0];
		expect(res.status).toBe('OK');
		expect(typeof res.duration.text).toBe('string');
		expect(typeof res.distance.text).toBe('string');
		expect(typeof res.duration.value).toBe('number');
		expect(typeof res.distance.value).toBe('number');
	});

	it('maps_elevation', async () => {
		let response = await Call(call, 'maps_elevation', {
			locations: [{ latitude: 48.2960451, longitude: 4.0651728 }],
		});
		let value = StandardDecode(response);
		expect(value.results.length).toBeGreaterThan(0);
		expect(typeof value.results[0].elevation).toBe('number');
		expect(typeof value.results[0].resolution).toBe('number');
		expect(typeof value.results[0].location.lat).toBe('number');
		expect(typeof value.results[0].location.lng).toBe('number');
	});

	it('maps_directions', async () => {
		let response = await Call(call, 'maps_directions', {
			origin: 'Troyes',
			destination: 'Paris',
		});
		let value = StandardDecode(response);
		expect(value.routes.length).toBeGreaterThan(0);
		let route = value.routes[0];
		expect(typeof route.summary).toBe('string');
		expect(typeof route.distance.text).toBe('string');
		expect(typeof route.duration.text).toBe('string');
		expect(typeof route.duration.value).toBe('number');
		expect(typeof route.distance.value).toBe('number');
		expect(route.steps.length).toBeGreaterThan(0);
	});
});
