import type {
  ApiErrorResponse,
  Assets,
  AuthResponse,
  ComposeResponse,
  DerivePromptResponse,
  Framework,
  FrameworkSuggestion,
  HealthResponse,
  LLMNormalizeResponse,
  NormalizedAssetsResponse,
  Prompt,
  PromptListResponse,
  PromptVersion,
  PromptVersionListResponse,
  QualityScore,
  RefineRequest,
  RefineResponse,
  Technique,
  ValidateSlotsResponse,
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

export async function listFrameworks(token: string): Promise<{ items: Framework[] }> {
  return request<{ items: Framework[] }>("/api/v1/frameworks", { token });
}

export async function listTechniques(token: string): Promise<{ items: Technique[] }> {
  return request<{ items: Technique[] }>("/api/v1/techniques", { token });
}

export async function normalizeAssets(
  token: string,
  assets: Assets,
): Promise<NormalizedAssetsResponse> {
  return request<NormalizedAssetsResponse>("/api/v1/assets/normalize", {
    method: "POST",
    token,
    body: { assets },
  });
}

export async function validateSlots(
  token: string,
  assets: Assets,
  frameworkId: string,
): Promise<ValidateSlotsResponse> {
  return request<ValidateSlotsResponse>("/api/v1/assets/validate", {
    method: "POST",
    token,
    body: { assets, frameworkId },
  });
}

export async function compose(
  token: string,
  input: { assets: Assets; frameworkId: string; techniqueIds: string[] },
): Promise<ComposeResponse> {
  return request<ComposeResponse>("/api/v1/compose", {
    method: "POST",
    token,
    body: input,
  });
}

export async function llmNormalize(
  token: string,
  answers: Record<string, string>,
): Promise<LLMNormalizeResponse> {
  return request<LLMNormalizeResponse>("/api/v1/llm/normalize", {
    method: "POST",
    token,
    body: { answers },
  });
}

export async function llmSuggestFramework(
  token: string,
  assets: Assets,
): Promise<FrameworkSuggestion> {
  return request<FrameworkSuggestion>("/api/v1/llm/suggest-framework", {
    method: "POST",
    token,
    body: { assets },
  });
}

export async function llmScore(
  token: string,
  composedOutput: string,
): Promise<QualityScore> {
  return request<QualityScore>("/api/v1/llm/score", {
    method: "POST",
    token,
    body: { composedOutput },
  });
}

export async function llmRefine(
  token: string,
  input: RefineRequest,
): Promise<RefineResponse> {
  return request<RefineResponse>("/api/v1/llm/refine", {
    method: "POST",
    token,
    body: input,
  });
}
