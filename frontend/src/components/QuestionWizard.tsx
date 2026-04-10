"use client";

import { useState } from "react";

type Step = {
  key: string;
  question: string;
  hint: string;
  required: boolean;
  rows: number;
  isExamples?: boolean;
};

const STEPS: Step[] = [
  {
    key: "goal",
    question: "What do you want this AI prompt to achieve?",
    hint: 'e.g. "Help customers resolve their support issues quickly and empathetically"',
    required: true,
    rows: 3,
  },
  {
    key: "persona",
    question: "Who should the AI be? Give it a role or personality.",
    hint: 'e.g. "A friendly customer support agent", "An experienced technical writer"',
    required: false,
    rows: 2,
  },
  {
    key: "context",
    question: "What background should the AI know before responding?",
    hint: 'e.g. "Users are small business owners with limited technical knowledge"',
    required: false,
    rows: 3,
  },
  {
    key: "tone",
    question: "How should the AI sound? Describe the tone or style.",
    hint: 'e.g. "Professional but approachable", "Casual and conversational"',
    required: false,
    rows: 2,
  },
  {
    key: "constraints",
    question: "What rules must the AI follow or avoid?",
    hint: 'e.g. "Never mention competitor products", "Always ask a follow-up question"',
    required: false,
    rows: 3,
  },
  {
    key: "examples",
    question: "Show me an example — what would a good input and output look like?",
    hint: "",
    required: false,
    rows: 2,
    isExamples: true,
  },
];

type Props = {
  onComplete: (answers: Record<string, string>) => void;
  onCancel: () => void;
  error?: string;
};

export default function QuestionWizard({ onComplete, onCancel, error }: Props) {
  const [currentStep, setCurrentStep] = useState(0);
  const [answers, setAnswers] = useState<Record<string, string>>({});
  const [exampleInput, setExampleInput] = useState("");
  const [exampleOutput, setExampleOutput] = useState("");

  const step = STEPS[currentStep];
  const isLastStep = currentStep === STEPS.length - 1;
  const isFirstStep = currentStep === 0;
  const progress = ((currentStep + 1) / STEPS.length) * 100;

  function handleAnswerChange(value: string) {
    setAnswers((prev) => ({ ...prev, [step.key]: value }));
  }

  function buildFinalAnswers(): Record<string, string> {
    const final = { ...answers };
    if (exampleInput.trim() || exampleOutput.trim()) {
      final.examples = `Input: ${exampleInput.trim()}\nOutput: ${exampleOutput.trim()}`;
    }
    return final;
  }

  function handleNext() {
    if (isLastStep) {
      onComplete(buildFinalAnswers());
    } else {
      setCurrentStep((s) => s + 1);
    }
  }

  function handleSkip() {
    if (isLastStep) {
      onComplete(buildFinalAnswers());
    } else {
      setCurrentStep((s) => s + 1);
    }
  }

  function handlePrev() {
    setCurrentStep((s) => s - 1);
  }

  const currentAnswer = step.isExamples
    ? exampleInput.trim() || exampleOutput.trim()
    : (answers[step.key] ?? "").trim();

  const canProceed = step.required ? currentAnswer.length > 0 : true;

  return (
    <div className="stack">
      <div className="row">
        <span className="muted" style={{ fontSize: "0.85rem" }}>
          Step {currentStep + 1} of {STEPS.length}
          {!step.required && (
            <span style={{ marginLeft: "0.4rem" }}>(optional)</span>
          )}
        </span>
        <button
          type="button"
          className="btn btn-secondary"
          style={{ fontSize: "0.8rem", padding: "0.3rem 0.7rem" }}
          onClick={onCancel}
        >
          Switch to manual editor
        </button>
      </div>

      <div
        style={{
          height: "4px",
          borderRadius: "2px",
          background: "var(--border)",
          overflow: "hidden",
        }}
      >
        <div
          style={{
            height: "100%",
            borderRadius: "2px",
            background: "var(--primary)",
            width: `${progress}%`,
            transition: "width 0.3s ease",
          }}
        />
      </div>

      {error && <p className="error">{error}</p>}

      <div className="card">
        <p style={{ fontWeight: 600, fontSize: "1.05rem", margin: "0 0 0.35rem" }}>
          {step.question}
        </p>
        {step.hint && (
          <p className="muted" style={{ fontSize: "0.85rem", margin: "0 0 0.9rem" }}>
            {step.hint}
          </p>
        )}

        {step.isExamples ? (
          <div className="stack">
            <label className="field" style={{ marginBottom: 0 }}>
              <span style={{ fontSize: "0.9rem" }}>Example input</span>
              <textarea
                className="textarea"
                rows={2}
                placeholder='e.g. "How do I cancel my subscription?"'
                value={exampleInput}
                onChange={(e) => setExampleInput(e.target.value)}
              />
            </label>
            <label className="field" style={{ marginBottom: 0 }}>
              <span style={{ fontSize: "0.9rem" }}>Example output</span>
              <textarea
                className="textarea"
                rows={2}
                placeholder='e.g. "Sure! Go to Settings → Billing → Cancel. Takes about 30 seconds."'
                value={exampleOutput}
                onChange={(e) => setExampleOutput(e.target.value)}
              />
            </label>
          </div>
        ) : (
          <textarea
            className="textarea"
            rows={step.rows}
            value={answers[step.key] ?? ""}
            onChange={(e) => handleAnswerChange(e.target.value)}
            autoFocus
          />
        )}
      </div>

      <div className="row">
        <button
          type="button"
          className="btn btn-secondary"
          onClick={handlePrev}
          disabled={isFirstStep}
        >
          Back
        </button>
        <div style={{ display: "flex", gap: "0.5rem" }}>
          {!step.required && (
            <button type="button" className="btn btn-secondary" onClick={handleSkip}>
              {isLastStep ? "Skip & finish" : "Skip"}
            </button>
          )}
          <button
            type="button"
            className="btn btn-primary"
            onClick={handleNext}
            disabled={!canProceed}
          >
            {isLastStep ? "Finish" : "Next"}
          </button>
        </div>
      </div>
    </div>
  );
}
