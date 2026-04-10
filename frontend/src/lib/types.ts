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

export type Assets = {
  persona: string;
  context: string;
  tone: string;
  constraints: string;
  examples: AssetExample[];
  goal: string;
};

export type AssetExample = {
  input: string;
  output: string;
};

export type FieldQuality = "empty" | "weak" | "complete";

export type NormalizedAssetsResponse = {
  assets: Assets;
  fieldReport: Record<string, FieldQuality>;
};

export type FrameworkSlot = {
  name: string;
  description: string;
  required: boolean;
  assetField: string;
};

export type Framework = {
  id: string;
  name: string;
  description: string;
  slots: FrameworkSlot[];
};

export type Technique = {
  id: string;
  name: string;
  description: string;
};

export type SlotMapping = {
  slotName: string;
  value: string;
  source: string;
};

export type DiagnosticSeverity = "error" | "warning";

export type SlotDiagnostic = {
  slotName: string;
  field: string;
  severity: DiagnosticSeverity;
  message: string;
};

export type ComposeResponse = {
  composedOutput: string;
  slotMap: SlotMapping[];
  diagnostics: SlotDiagnostic[];
  frameworkId: string;
  techniqueIds: string[];
};

export type ValidateSlotsResponse = {
  diagnostics: SlotDiagnostic[];
  hasErrors: boolean;
};
