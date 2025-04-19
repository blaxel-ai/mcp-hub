export const payload: ((previousResult: Record<string, any>) => {
  name: string;
  arguments: Record<string, any>;
})[] = [
  () => ({
    name: "ask_user",
    arguments: {
      question: "What is the meaning of life?",
    },
  }),
];

export const description = "Telegram description";
export const name = "telegram";
export const url = "http://localhost:8080";
