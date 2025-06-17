export const payload: ((previousResult: Record<string, any>) => {
  name: string;
  arguments: Record<string, any>;
})[] = [
  () => ({
    name: "send-email",
    arguments: {
      to: "cploujoux@blaxel.ai",
      subject: "Test Email",
      text: "This is a test email",
      from: "onboarding@resend.dev"
    },
  }),
];

export const description = "Resend Toolkit description";
export const name = "resend";
export const url = "http://localhost:1400";
