import { describe, expect, it } from 'vitest';

import { call, list } from '../src/linear';
import { Call, DefineSecret, StandardDecode } from './beamlit';


let silent = true;

describe('Linear', async() => {
	// @ts-ignore
	const apiKey = import.meta.env.LINEAR_API_KEY
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
      'add_comment',
      'create_issue',
      'get_issue_details',
      'list_teams',
      'list_users',
			'search_issues',
      'update_issue',
      'get_user_issues',
		]
    for(let func of listFunc) {
      expect(tools.some((tool) => tool.name === func),`${func} is missing`).toBe(true);
    }
	})

  it('list_users', async () => {
    let response = await Call(call, 'list_users', {})
    let value = StandardDecode(response)
    expect(value.length).toBeGreaterThan(0);
  })

  it('create_issue', async () => {
    let response = await Call(call, 'create_issue', {
      title: 'Test Issue',
      teamKey: 'CHA',
      description: 'This is a test issue',
      assigneeEmail: 'cdrappier@beamlit.com',
      status: 'In Progress',
      priority: 2,
    })
    let value = StandardDecode(response)
    expect(value.id).toBeDefined();
  },10000)

  it ('update_issue', async () => {
    let response = await Call(call, 'update_issue', {
      issueKey: 'CHA-52',
      title: 'CA ZAGOUILLE',
      description: 'This is a test issue changed',
      assigneeEmail: 'cdrappier@beamlit.com',
      status: 'In Progress',
      priority: 2,
    })
    let value = StandardDecode(response)
    expect(value.id).toBeDefined();
  },10000)

  it('add_comment', async () => {
    let response = await Call(call, 'add_comment', {
      issueKey: 'CHA-52',
      comment: 'This is a test comment',
      userEmail: 'cdrappier@beamlit.com',
    })
    let value = StandardDecode(response)
    expect(value.id).toBeDefined();
  })

	it('search_issues', async () => {
		let response = await Call(call, 'search_issues', {
			query: 'Test Issue',
      teamKey: 'CHA',
      // status: 'Todo',
      // assigneeEmail: 'cdrappier@beamlit.com',
      // assigneeName: 'Charles Drappier',
      // labels: ['Bug'],
      // priority: 2,
      // estimate: null,
      // includeArchived: false,
      // limit: 2,
		})
		let value = StandardDecode(response)[0]
		expect(value.id,'Issue ID is missing').toBeDefined();
    expect(value.title,'Issue title is missing').toBeDefined();
    expect(value.description,'Issue description is missing').toBeDefined();
    expect(value.priority,'Issue priority is missing').toBeDefined();
    expect(value.estimate,'Issue estimate is missing').toBeDefined();
	})

  it('get_issue_details', async () => {
    let response = await Call(call, 'get_issue_details', {
      issueKey: 'CHA-25',
    })
    let value = StandardDecode(response)
    expect(value.id).toBeDefined();
    expect(value.title).toBeDefined();
    expect(value.description).toBeDefined();
  })

  it('list_teams', async () => {
    let response = await Call(call, 'list_teams', {})
    let value = StandardDecode(response)
    expect(value.length).toBeGreaterThan(0);
  })

  it('get_user_issues', async () => {
    let response = await Call(call, 'get_user_issues', {
      userEmail: 'cdrappier@beamlit.com',
    })
    let value = StandardDecode(response)
    expect(value.length).toBeGreaterThan(0);
  })


});
