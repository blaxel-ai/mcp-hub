import { AddCommentArgs } from "./addComment.js";
import { CreateIssueArgs } from "./createIssue.js";
import { GetIssueDetailsArgs } from "./getIssueDetails.js";
import { GetUserIssuesArgs } from "./getUserIssues.js";
import { ListTeamsArgs } from "./listTeams.js";
import { ListUsersArgs } from "./listUsers.js";
import { SearchIssuesArgs } from "./searchIssues.js";
import { UpdateIssueArgs } from "./updateIssue.js";
export class LinearMCPClient {
  private baseUrl: string;
  private headers: Record<string, string>;
  constructor(config: Record<string, string>, secrets: Record<string, string>) {
    this.baseUrl = 'https://api.linear.app/graphql';
    this.headers = {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${secrets.apiKey}`,
    };
  }

  private parseError(json: any) {
    if (json.errors) {
      console.error(json.errors[0])
      let error = new Error()
      error.message = json.errors.map((e: any) => `${e.message} - ${JSON.stringify(e)}`).join('\n\n');
      return error;
    }
    return null;
  }

  async fetch(query: string) {
    const response = await fetch(this.baseUrl, {
      method: 'POST',
      headers: this.headers,
      body: JSON.stringify({ query }),
    });
    let json:any = await response.json()
    let error = this.parseError(json)
    if (error) {
      throw new Error(`Error fetching issues: ${error.message}`);
    }
    return json
  }


  async addComment(args: AddCommentArgs) {
    const response:any = await this.fetch(`mutation {
      commentCreate(
        input: {
          issueId: "${args.issueKey}",
          body: "${args.comment}"
          ${args.userEmail ? `createAsUser: "${args.userEmail}"` : ''}
        }
      ) {
        success
        comment {
          id
        }
      }
    }`);
    return {
      content: [{ type: 'text', text: JSON.stringify(response.data.commentCreate.comment, null, 2) }],
      isError: false,
    };
  }

  async listUsers(args: ListUsersArgs) {
    const response:any = await this.fetch(`query {
      users {
        nodes {
          id
          name
          email
        }
      }
    }`);
    return {
      content: [{ type: 'text', text: JSON.stringify(response.data.users.nodes, null, 2) }],
      isError: false,
    };
  }

  async listTeams(args: ListTeamsArgs) {
    const response:any = await this.fetch(`query {
      teams {
        nodes {
          id
          name
          key
          displayName
        }
      }
    }`);
    return {
      content: [{ type: 'text', text: JSON.stringify(response.data.teams.nodes, null, 2) }],
      isError: false,
    };
  }

  async getUserByEmail(email: string) {
    const response:any = await this.fetch(`query {
      users(filter: {email: {eq: "${email}"}}) {
        nodes {
          id
        }
      }
    }`);
    return response.data.users.nodes[0].id
  }

  async getStates() {
    const response:any = await this.fetch(`query {
      teams {
        nodes {
          id
          states {
            nodes {
              id
              name
            }
          }
        }
      }
    }`);
    let states: Record<string, string> = {}
    for(let team of response.data.teams.nodes) {
      for(let state of team.states.nodes) {
        states[state.name] = state.id
      }
    }
    return states
  }

  async getStateId(name: string) {
    const states = await this.getStates()
    return states[name]
  }

  async updateIssue(args: UpdateIssueArgs) {
    let stateId = null
    if (args.status) {
      stateId = await this.getStateId(args.status)
    }
    let assigneeId = null
    // convert email to id
    if (args.assigneeEmail) {
      assigneeId = await this.getUserByEmail(args.assigneeEmail)
    }
    const response:any = await this.fetch(`mutation {
      issueUpdate(
        id: "${args.issueKey}",
        input: {
          ${args.title ? `title: "${args.title}"` : ''}
          ${args.description ? `description: "${args.description}"` : ''}
          ${stateId ? `stateId: "${stateId}"` : ''}
          ${args.priority ? `priority: ${args.priority}` : ''}
          ${assigneeId ? `assigneeId: "${assigneeId}"` : ''}
        }
      ) {
        success
        issue {
          id
          title
          priority
          description
          assignee {
            id
            name
            email
          }
          state {
            id
            name
          }
        }
      }
    }`);
    return {
      content: [{ type: 'text', text: JSON.stringify(response.data.issueUpdate.issue, null, 2) }],
      isError: false,
    };
  }

  async getTeamByKey(key: string) {
    const response:any = await this.fetch(`query {
      teams(filter: {key: {eq: "${key}"}}) {
        nodes {
          id
        }
      }
    }`);
    return response.data.teams.nodes[0].id
  }

  async createIssue(args: CreateIssueArgs) {
    let assigneeId = null
    // convert email to id
    if (args.assigneeEmail) {
      assigneeId = await this.getUserByEmail(args.assigneeEmail)
    }
    let stateId = null
    if (args.status) {
      stateId = await this.getStateId(args.status)
    }
    let teamId = await this.getTeamByKey(args.teamKey)
    const response:any = await this.fetch(`mutation {
      issueCreate(
        input: {
          ${args.title ? `title: "${args.title}"` : ''}
          ${args.description ? `description: "${args.description}"` : ''}
          ${teamId ? `teamId: "${teamId}"` : ''}
          ${assigneeId ? `assigneeId: "${assigneeId}"` : ''}
          ${stateId ? `stateId: "${stateId}"` : ''}
          ${args.priority ? `priority: ${args.priority}` : ''}
        }
      ) {
        success
        issue {
          id
          title
        }
      }
    }`);
    return {
      content: [{ type: 'text', text: JSON.stringify(response.data.issueCreate.issue, null, 2) }],
      isError: false,
    };
  }

  async getIssueDetails(args: GetIssueDetailsArgs) {
    const response:any = await this.fetch(`query {
      issue(id: "${args.issueKey}") {
        id
        title
        description
      }
    }`);
    return {
      content: [{ type: 'text', text: JSON.stringify(response.data.issue, null, 2) }],
      isError: false,
    };
  }

  async getUserIssues(args: GetUserIssuesArgs) {
    const response:any = await this.fetch(`query {
      issues(filter: {assignee: {email: {eq: "${args.userEmail}"}}}) {
        nodes {
          id
          title
          description
          state {
            name
          }
          assignee {
            name
            email
          }
          team {
            key
          }
          assignee {
            name
            email
          }
          priority
          estimate
        }
      }
    }`);
    return {
      content: [{ type: 'text', text: JSON.stringify(response.data.issues.nodes, null, 2) }],
      isError: false,
    };
  }

  private buildSearchFilter(args: SearchIssuesArgs): any {
    const filter: any = {};
    if (args.query) {
      filter.or = [
        { title: { contains: args.query } },
        { description: { contains: args.query } }
      ];
    }
    if (args.teamKey) {
      filter.team = { key: { eq: args.teamKey } };
    }
    if (args.status) {
      filter.state = { name: { eq: args.status } };
    }
    if (args.assigneeEmail) {
      filter.assignee = filter.assignee || {};
      filter.assignee.email = { eq: args.assigneeEmail };
    }
    if (args.assigneeName) {
      filter.assignee = filter.assignee || {};
      filter.assignee.name = { eq: args.assigneeName };
    }
    if (args.labels && args.labels.length > 0) {
      filter.labels = {
        some: {
          name: { in: args.labels }
        }
      };
    }
    if (args.priority) {
      filter.priority = { eq: args.priority };
    }
    if (args.estimate) {
      filter.estimate = { eq: args.estimate };
    }
    return filter;
  }

  async searchIssues(args: SearchIssuesArgs) {
    // Construct filter object with only defined values
    const filterArgs = this.buildSearchFilter(args)
    const response : any = await this.fetch(`query {
      issues(
        filter: ${JSON.stringify(filterArgs).replace(/"([^"]+)":/g, '$1:')}
        first: ${args.limit || 10}
      ) {
        nodes {
          id
          title
          description
          state {
            name
          }
          assignee {
            name
            email
          }
          team {
            key
          }
          assignee {
            name
            email
          }
          priority
          estimate
        }
      }
    }`);

    if (response.errors) {
      throw new Error(`Error fetching issues: ${response.errors.map((e: any) => e.message).join(', ')}`);
    }


    return {
      content: [{ type: 'text', text: JSON.stringify(response.data.issues.nodes, null, 2) }],
      isError: false,
    };
  }
}
