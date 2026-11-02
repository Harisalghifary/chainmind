import { create } from "zustand";

import {
  ApiError,
  getAuditReport,
  getAuditStatus,
  submitAudit,
} from "@/lib/api";
import type {
  AuditReport,
  AuditStatusResponse,
  CreateAuditRequest,
} from "@/lib/types";

interface AuditStore {
  status: AuditStatusResponse | null;
  report: AuditReport | null;
  isLoading: boolean;
  isSubmitting: boolean;
  error: string | null;
  submitError: string | null;
  submitAudit: (input: CreateAuditRequest) => Promise<string>;
  startPolling: (auditId: string) => void;
  stopPolling: () => void;
  reset: () => void;
}

let pollTimer: ReturnType<typeof setInterval> | null = null;
let pollGeneration = 0;

function stopTimer() {
  if (pollTimer) {
    clearInterval(pollTimer);
    pollTimer = null;
  }
}

function errorMessage(error: unknown): string {
  if (error instanceof ApiError) return error.message;
  if (error instanceof Error) return error.message;
  return "Something went wrong.";
}

export const useAuditStore = create<AuditStore>((set, get) => ({
  status: null,
  report: null,
  isLoading: false,
  isSubmitting: false,
  error: null,
  submitError: null,

  async submitAudit(input) {
    set({ isSubmitting: true, submitError: null });
    try {
      const created = await submitAudit(input);
      set({ isSubmitting: false });
      return created.audit_id;
    } catch (error) {
      set({ isSubmitting: false, submitError: errorMessage(error) });
      throw error;
    }
  },

  startPolling(auditId) {
    stopTimer();
    pollGeneration += 1;
    const generation = pollGeneration;

    set({
      status: null,
      report: null,
      error: null,
      isLoading: true,
    });

    const tick = async () => {
      try {
        const status = await getAuditStatus(auditId);
        if (generation !== pollGeneration) return;

        set({ status, isLoading: false });

        if (status.status === "COMPLETED") {
          const report = await getAuditReport(auditId);
          if (generation !== pollGeneration) return;
          set({ report });
          get().stopPolling();
          return;
        }

        if (status.status === "FAILED") {
          get().stopPolling();
        }
      } catch (error) {
        if (generation !== pollGeneration) return;
        set({ error: errorMessage(error), isLoading: false });
        get().stopPolling();
      }
    };

    pollTimer = setInterval(() => {
      void tick();
    }, 1500);

    void tick();
  },

  stopPolling() {
    pollGeneration += 1;
    stopTimer();
  },

  reset() {
    get().stopPolling();
    set({
      status: null,
      report: null,
      isLoading: false,
      isSubmitting: false,
      error: null,
      submitError: null,
    });
  },
}));