import { describe, expect, it } from 'vitest';
import { call, list } from '../src/dall-e';

import { Call, DefineSecret, StandardDecode } from './blaxel';


let silent = true;

describe('Dall-E', async() => {
	// @ts-ignore
	const apiKey = import.meta.env.OPENAI_LOCAL_KEY
	DefineSecret('apiKey', apiKey)

	it('listTools', async () => {
		const {tools} = await list();
		expect(tools).toBeDefined();
		expect(tools.length).toBeGreaterThan(0);
		for(let tool of tools) {
			expect(tool.name).toBeDefined();
			expect(tool.description).toBeDefined();
			expect(tool.inputSchema).toBeDefined();
		}
		let listFunc = [
			'generate_image',
		]
		// asert it's inside the list
		expect(listFunc.every((func) => tools.some((tool) => tool.name === func))).toBe(true);
	})

	it('generate_image', async () => {
		let response = await Call(call, 'generate_image', {
			prompt: 'A tiger in the snow',
			size: '1024x1024',
			imageName: 'tiger'
		})
		console.log(response)
		let value = StandardDecode(response)
		expect(value).toBeDefined();
	}, 60000)
});
