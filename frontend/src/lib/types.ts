export type ApiErrorResponse = {
  error: string;
};

export type User = {
  id: string;
  email: string;
  createdAt: string;
};

export type AuthResponse = {
  user: User;
  token: string;
};

export type Prompt = {
  id: string;
  title: string;
  status: string;
  category: string;
  tags: string[];
  ownerId: string;
  createdAt: string;
  updatedAt: string;
};

export type PromptVersion = {
  id: string;
  promptId: string;
  versionNumber: number;
  assets: unknown;
  frameworkId: string;
  techniqueIds: string[];
  composedOutput: string;
  createdAt: string;
};

export type PromptListResponse = {
  items: Prompt[];
};

export type PromptVersionListResponse = {
  items: PromptVersion[];
};

export type DerivePromptResponse = {
  prompt: Prompt;
  version: PromptVersion;
};

export type HealthResponse = {
  status: string;
};
