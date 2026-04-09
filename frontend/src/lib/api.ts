import type {
  ApiErrorResponse,
  AuthResponse,
  DerivePromptResponse,
  HealthResponse,
  Prompt,
  PromptListResponse,
  PromptVersion,
  PromptVersionListResponse,
} from "@/lib/types";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? "/backend";

export class ApiError extends Error {
  status: number;

  constructor(message: string, status: number) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

type RequestOptions = {
  method?: "GET" | "POST" | "PATCH";
  body?: unknown;
  token?: string | null;
};

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const headers = new Headers();
  headers.set("Content-Type", "application/json");

  if (options.token) {
    headers.set("Authorization", `Bearer ${options.token}`);
  }

  let response: Response;
  try {
    response = await fetch(`${API_BASE_URL}${path}`, {
      method: options.method ?? "GET",
      headers,
      body: options.body ? JSON.stringify(options.body) : undefined,
    });
  } catch {
    throw new ApiError("Network error: cannot reach API server", 0);
  }

  const isJson = response.headers.get("content-type")?.includes("application/json");
  const payload = isJson ? ((await response.json()) as unknown) : null;

  if (!response.ok) {
    const fallbackMessage = `Request failed with status ${response.status}`;
    if (payload && typeof payload === "object" && "error" in payload) {
      const apiErr = payload as ApiErrorResponse;
      throw new ApiError(apiErr.error || fallbackMessage, response.status);
    }
    throw new ApiError(fallbackMessage, response.status);
  }

  return payload as T;
}

export async function getHealth(): Promise<HealthResponse> {
  return request<HealthResponse>("/health");
}

export async function register(input: {
  email: string;
  password: string;
}): Promise<AuthResponse> {
  return request<AuthResponse>("/api/v1/auth/register", {
    method: "POST",
    body: input,
  });
}

export async function login(input: {
  email: string;
  password: string;
}): Promise<AuthResponse> {
  return request<AuthResponse>("/api/v1/auth/login", {
    method: "POST",
    body: input,
  });
}

export async function listPrompts(token: string): Promise<PromptListResponse> {
  return request<PromptListResponse>("/api/v1/prompts", { token });
}

export async function createPrompt(
  token: string,
  input: {
    title: string;
    status: string;
    category: string;
    tags: string[];
  },
): Promise<Prompt> {
  return request<Prompt>("/api/v1/prompts", {
    method: "POST",
    token,
    body: input,
  });
}

export async function getPrompt(token: string, promptId: string): Promise<Prompt> {
  return request<Prompt>(`/api/v1/prompts/${promptId}`, { token });
}

export async function updatePrompt(
  token: string,
  promptId: string,
  input: {
    title: string;
    status: string;
    category: string;
    tags: string[];
  },
): Promise<Prompt> {
  return request<Prompt>(`/api/v1/prompts/${promptId}`, {
    method: "PATCH",
    token,
    body: input,
  });
}

export async function listPromptVersions(token: string, promptId: string): Promise<PromptVersionListResponse> {
  return request<PromptVersionListResponse>(`/api/v1/prompts/${promptId}/versions`, { token });
}

export async function createPromptVersion(
  token: string,
  promptId: string,
  input: {
    assets: unknown;
    frameworkId: string;
    techniqueIds: string[];
    composedOutput: string;
  },
): Promise<PromptVersion> {
  return request<PromptVersion>(`/api/v1/prompts/${promptId}/versions`, {
    method: "POST",
    token,
    body: input,
  });
}

export async function derivePrompt(
  token: string,
  input: {
    sourcePromptId: string;
    sourceVersionId?: string;
    newTitle: string;
  },
): Promise<DerivePromptResponse> {
  return request<DerivePromptResponse>("/api/v1/prompts/derive", {
    method: "POST",
    token,
    body: input,
  });
}
