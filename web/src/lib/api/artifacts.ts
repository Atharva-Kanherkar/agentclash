import { ApiError, NetworkError } from "./errors";
import type {
  ArtifactUploadResponse,
  ArtifactDownloadResponse,
  ApiErrorResponse,
} from "./types";

function resolveBaseUrl(): string {
  const url =
    typeof window === "undefined"
      ? process.env.API_URL ?? process.env.NEXT_PUBLIC_API_URL
      : process.env.NEXT_PUBLIC_API_URL;
  if (!url) {
    throw new Error(
      "Missing API_URL or NEXT_PUBLIC_API_URL environment variable",
    );
  }
  return url.replace(/\/+$/, "");
}

export interface UploadArtifactParams {
  token: string;
  workspaceId: string;
  file: File;
  artifactType: string;
  runId?: string;
  runAgentId?: string;
  metadata?: Record<string, unknown>;
  onProgress?: (percent: number) => void;
}

/** Upload an artifact via multipart form POST. */
export async function uploadArtifact(
  params: UploadArtifactParams,
): Promise<ArtifactUploadResponse> {
  const {
    token,
    workspaceId,
    file,
    artifactType,
    runId,
    runAgentId,
    metadata,
    onProgress,
  } = params;

  const form = new FormData();
  form.append("file", file);
  form.append("artifact_type", artifactType);
  if (runId) form.append("run_id", runId);
  if (runAgentId) form.append("run_agent_id", runAgentId);
  if (metadata) form.append("metadata", JSON.stringify(metadata));

  const url = `${resolveBaseUrl()}/v1/workspaces/${workspaceId}/artifacts`;

  // Use XMLHttpRequest for progress tracking
  if (onProgress) {
    return new Promise<ArtifactUploadResponse>((resolve, reject) => {
      const xhr = new XMLHttpRequest();
      xhr.open("POST", url);
      xhr.setRequestHeader("Authorization", `Bearer ${token}`);

      xhr.upload.onprogress = (e) => {
        if (e.lengthComputable) {
          onProgress(Math.round((e.loaded / e.total) * 100));
        }
      };

      xhr.onload = () => {
        try {
          const body = JSON.parse(xhr.responseText);
          if (xhr.status === 201) {
            resolve(body as ArtifactUploadResponse);
          } else {
            const err = body as ApiErrorResponse;
            reject(
              new ApiError(
                xhr.status,
                err.error?.code ?? "unknown",
                err.error?.message ?? "Upload failed",
              ),
            );
          }
        } catch {
          reject(new ApiError(xhr.status, "unknown", "Upload failed"));
        }
      };

      xhr.onerror = () => {
        reject(new NetworkError("Network request failed"));
      };

      xhr.send(form);
    });
  }

  // Simple fetch for cases without progress
  let res: Response;
  try {
    res = await fetch(url, {
      method: "POST",
      headers: { Authorization: `Bearer ${token}` },
      body: form,
    });
  } catch (err) {
    throw new NetworkError(
      err instanceof Error ? err.message : "Network request failed",
    );
  }

  if (!res.ok) {
    try {
      const body = (await res.json()) as ApiErrorResponse;
      throw new ApiError(res.status, body.error.code, body.error.message);
    } catch (e) {
      if (e instanceof ApiError) throw e;
      throw new ApiError(res.status, "unknown", "Upload failed");
    }
  }

  return res.json() as Promise<ArtifactUploadResponse>;
}

/** Fetch a signed download URL and open it in a new tab. */
export async function downloadArtifact(
  token: string,
  artifactId: string,
): Promise<void> {
  const url = `${resolveBaseUrl()}/v1/artifacts/${artifactId}/download`;

  let res: Response;
  try {
    res = await fetch(url, {
      headers: {
        Authorization: `Bearer ${token}`,
        Accept: "application/json",
      },
    });
  } catch (err) {
    throw new NetworkError(
      err instanceof Error ? err.message : "Network request failed",
    );
  }

  if (!res.ok) {
    try {
      const body = (await res.json()) as ApiErrorResponse;
      throw new ApiError(res.status, body.error.code, body.error.message);
    } catch (e) {
      if (e instanceof ApiError) throw e;
      throw new ApiError(res.status, "unknown", "Download failed");
    }
  }

  const data = (await res.json()) as ArtifactDownloadResponse;
  window.open(data.url, "_blank");
}
