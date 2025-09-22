export const payload: ((previousResult: Record<string, any>) => {
  name: string;
  arguments: Record<string, any>;
})[] = [
  () => ({
    name: "list_inboxes",
    arguments: {
    },
  }),

];

export const description = "AgentMail Toolkit description";
export const name = "agentmail";
export const url = "http://localhost:1400";
