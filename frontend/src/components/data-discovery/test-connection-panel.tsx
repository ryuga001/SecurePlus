"use client";

import { CircleCheck, Loader2, PlugZap, TriangleAlert } from "lucide-react";
import { useTranslations } from "next-intl";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";

export type SaveStage = "idle" | "validating" | "testing" | "saving";

export type TestState =
  | { status: "idle" }
  | { status: "running" }
  | { status: "passed" }
  | { status: "failed"; message: string; code?: string };

const STAGES: Exclude<SaveStage, "idle">[] = ["validating", "testing", "saving"];

export function TestConnectionPanel({
  stage,
  test,
  canTest,
  onTest,
}: {
  stage: SaveStage;
  test: TestState;
  canTest: boolean;
  onTest: () => void;
}) {
  const t = useTranslations("data-discovery.configurations.dialog");

  const running = test.status === "running" || stage === "testing";

  return (
    <div className="flex flex-col gap-3 rounded-lg border p-4">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <h3 className="text-sm font-semibold">{t("connectionTitle")}</h3>
          <p className="mt-1 text-xs text-muted-foreground">{t("connectionHint")}</p>
        </div>

        <Button
          type="button"
          variant="outline"
          size="sm"
          disabled={!canTest || running || stage !== "idle"}
          onClick={onTest}
        >
          {running ? <Loader2 className="animate-spin" /> : <PlugZap />}
          {test.status === "passed" ? t("testAgain") : t("testConnection")}
        </Button>
      </div>

      {test.status === "passed" ? (
        <div className="flex items-center gap-2 rounded-md border border-success-border bg-success-container px-3 py-2 text-sm text-success-text">
          <CircleCheck className="size-4 shrink-0" />
          {t("connectionVerified")}
        </div>
      ) : null}

      {test.status === "failed" ? (
        <div className="flex items-start gap-2 rounded-md border border-error-border bg-error-container px-3 py-2 text-sm text-error-text">
          <TriangleAlert className="mt-0.5 size-4 shrink-0" />
          <div className="min-w-0 flex-1">
            <p>{test.message}</p>
            {test.code ? (
              <Badge variant="error" size="sm" className="mt-1.5 font-mono">
                {test.code}
              </Badge>
            ) : null}
          </div>
        </div>
      ) : null}

      {stage !== "idle" ? (
        <div aria-live="polite" className="flex flex-wrap items-center gap-4 text-xs">
          {STAGES.map((item) => {
            const current = stage === item;
            const done = STAGES.indexOf(stage as Exclude<SaveStage, "idle">) > STAGES.indexOf(item);

            return (
              <span
                key={item}
                className={
                  current
                    ? "flex items-center gap-1.5 font-medium text-foreground"
                    : done
                      ? "flex items-center gap-1.5 text-success-text"
                      : "flex items-center gap-1.5 text-muted-foreground"
                }
              >
                {current ? (
                  <Loader2 className="size-3 animate-spin" />
                ) : done ? (
                  <CircleCheck className="size-3" />
                ) : (
                  <span className="size-3" />
                )}
                {t(`stage.${item}`)}
              </span>
            );
          })}
        </div>
      ) : null}
    </div>
  );
}
