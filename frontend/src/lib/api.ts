import type {
  AuditCreated,
  AuditReport,
  AuditStatusResponse,
  CreateAuditRequest,
  ExploitListResponse,
} from "./types";

const API_URL =
  process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080/api/v1";

export class ApiError extends Error {
  readonly code: string;
  readonly status: number;

  constructor(message: string, code: string, status: number) {
    super(message);
    this.name = "ApiError";
    this.code = code;
    this.status = status;
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  let response: Response;

  try {
    response = await fetch(`${API_URL}${path}`, {
      cache: "no-store",
      ...init,
      headers: {
        "Content-Type": "application/json",
        ...(init?.headers ?? {}),
      },
    });
  } catch {
    throw new ApiError(
      "Cannot reach the ChainMind API. Is the backend running?",
      "NETWORK_ERROR",
      0,
    );
  }

  if (!response.ok) {
    let code = "INTERNAL_ERROR";
    let message = `Request failed with status ${response.status}`;

    try {
      const body = (await response.json()) as {
        error?: string;
        message?: string;
      };
      code = body.error ?? code;
      message = body.message ?? message;
    } catch {
      // Keep the fallback message when the body is not JSON.
    }

    throw new ApiError(message, code, response.status);
  }

  return (await response.json()) as T;
}

export function submitAudit(
  payload: CreateAuditRequest,
): Promise<AuditCreated> {
  return request<AuditCreated>("/audit", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export function getAuditStatus(id: string): Promise<AuditStatusResponse> {
  return request<AuditStatusResponse>(`/audit/${encodeURIComponent(id)}`);
}

export function getAuditReport(id: string): Promise<AuditReport> {
  return request<AuditReport>(
    `/audit/${encodeURIComponent(id)}/report`,
  );
}

export function listExploits(params?: {
  category?: string;
  severity?: string;
  page?: number;
  limit?: number;
}): Promise<ExploitListResponse> {
  const query = new URLSearchParams();
  if (params?.category) query.set("category", params.category);
  if (params?.severity) query.set("severity", params.severity);
  if (params?.page) query.set("page", String(params.page));
  if (params?.limit) query.set("limit", String(params.limit));

  const suffix = query.toString();
  return request<ExploitListResponse>(
    `/exploits${suffix ? `?${suffix}` : ""}`,
  );
}