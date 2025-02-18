import { describe, expect, it } from 'vitest';
import { call, list } from '../src/gmail';

import { Call, DefineSecret, StandardDecode } from './baxel';


let silent = true;

describe('Gmail', async() => {
  // @ts-ignore
  let refreshToken = import.meta.env.GOOGLE_REFRESH_TOKEN
	DefineSecret('refreshToken', refreshToken)

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
			'send_email',
		]
		// asert it's inside the list
		expect(listFunc.every((func) => tools.some((tool) => tool.name === func))).toBe(true);
	})

	it('send_email', async () => {
		let response = await Call(call, 'send_email', {
      "to": "cdrappier@baxel.com",
      "subject": "Voici une œuvre d'art féline",
      "body": "Bonjour,\n\nJe vous envoie une image artistique d'un chat que j'ai générée. J'espère qu'elle vous plaira !\n\nVoici le lien vers l'image : https://oaidalleapiprodscus.blob.core.windows.net/private/org-R54BJY18pbqSztZg68sSM5gV/sandbox/img-qI7TyzXFEHtoOA6v50cZkVSY.png\n\nCordialement,\n\nVotre assistant"
		})
    console.log(response);
		let value = StandardDecode(response)
    console.log(value);
	})
});
