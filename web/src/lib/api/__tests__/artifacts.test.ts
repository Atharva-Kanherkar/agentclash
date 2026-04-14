import { describe, it, expect, vi, beforeEach } from "vitest";
import { uploadArtifact, downloadArtifact } from "../artifacts";
import { ApiError, NetworkError } from "../errors";

vi.stubEnv("NEXT_PUBLIC_API_URL", "http://localhost:8080");

const mockFetch = vi.fn();
vi.stubGlobal("fetch", mockFetch);

// Mock XMLHttpRequest for upload progress tests
class MockXHR {
  open = vi.fn();
  send = vi.fn();
  setRequestHeader = vi.fn();
  status = 201;
  responseText = "{}";
  upload = { onprogress: null as ((e: unknown) => void) | null };
  onload: (() => void) | null = null;
  onerror: (() => void) | null = null;
}

let mockXHRInstance: MockXHR;

beforeEach(() => {
  mockFetch.mockReset();
  mockXHRInstance = new MockXHR();
  // Use a real constructor function so `new XMLHttpRequest()` works
  globalThis.XMLHttpRequest = function () {
    return mockXHRInstance;
  } as unknown as typeof XMLHttpRequest;
});

function jsonResponse(data: unknown, status = 200) {
  return new Response(JSON.stringify(data), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

function errorResponse(status: number, code: string, message: string) {
  return new Response(JSON.stringify({ error: { code, message } }), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

describe("uploadArtifact", () => {
  it("sends multipart form POST without progress callback", async () => {
    const uploadResponse = {
      id: "art-1",
      workspace_id: "ws-1",
      artifact_type: "log",
      visibility: "private",
      metadata: { original_filename: "test.txt" },
      created_at: "2026-04-13T00:00:00Z",
    };
    mockFetch.mockResolvedValueOnce(
      new Response(JSON.stringify(uploadResponse), { status: 201 }),
    );

    const file = new File(["hello"], "test.txt", { type: "text/plain" });
    const result = await uploadArtifact({
      token: "my-token",
      workspaceId: "ws-1",
      file,
      artifactType: "log",
    });

    expect(mockFetch).toHaveBeenCalledWith(
      "http://localhost:8080/v1/workspaces/ws-1/artifacts",
      expect.objectContaining({
        method: "POST",
        headers: expect.objectContaining({
          Authorization: "Bearer my-token",
        }),
      }),
    );
    expect(result.id).toBe("art-1");
    expect(result.artifact_type).toBe("log");
  });

  it("includes run_id and run_agent_id in form data", async () => {
    mockFetch.mockResolvedValueOnce(
      new Response(
        JSON.stringify({ id: "art-2", artifact_type: "output", visibility: "private", metadata: {}, created_at: "" }),
        { status: 201 },
      ),
    );

    const file = new File(["data"], "out.json", { type: "application/json" });
    await uploadArtifact({
      token: "tok",
      workspaceId: "ws-1",
      file,
      artifactType: "output",
      runId: "run-1",
      runAgentId: "agent-1",
      metadata: { note: "test" },
    });

    const body = mockFetch.mock.calls[0][1].body as FormData;
    expect(body.get("artifact_type")).toBe("output");
    expect(body.get("run_id")).toBe("run-1");
    expect(body.get("run_agent_id")).toBe("agent-1");
    expect(body.get("metadata")).toBe('{"note":"test"}');
  });

  it("throws ApiError on non-201 response", async () => {
    mockFetch.mockResolvedValueOnce(
      errorResponse(400, "invalid_artifact_type", "bad type"),
    );

    const file = new File(["x"], "f.bin");
    await expect(
      uploadArtifact({
        token: "tok",
        workspaceId: "ws-1",
        file,
        artifactType: "BAD",
      }),
    ).rejects.toThrow(ApiError);
  });

  it("throws NetworkError when fetch fails", async () => {
    mockFetch.mockRejectedValueOnce(new TypeError("Failed to fetch"));

    const file = new File(["x"], "f.bin");
    await expect(
      uploadArtifact({
        token: "tok",
        workspaceId: "ws-1",
        file,
        artifactType: "log",
      }),
    ).rejects.toThrow(NetworkError);
  });

  it("uses XMLHttpRequest when onProgress is provided", async () => {
    const progressFn = vi.fn();
    const file = new File(["hello"], "test.txt");

    const promise = uploadArtifact({
      token: "my-token",
      workspaceId: "ws-1",
      file,
      artifactType: "log",
      onProgress: progressFn,
    });

    // Simulate XHR behavior
    expect(mockXHRInstance.open).toHaveBeenCalledWith(
      "POST",
      "http://localhost:8080/v1/workspaces/ws-1/artifacts",
    );
    expect(mockXHRInstance.setRequestHeader).toHaveBeenCalledWith(
      "Authorization",
      "Bearer my-token",
    );

    // Simulate progress event
    mockXHRInstance.upload.onprogress?.({
      lengthComputable: true,
      loaded: 50,
      total: 100,
    });
    expect(progressFn).toHaveBeenCalledWith(50);

    // Simulate success
    mockXHRInstance.status = 201;
    mockXHRInstance.responseText = JSON.stringify({
      id: "art-3",
      artifact_type: "log",
      visibility: "private",
      metadata: {},
      created_at: "",
    });
    mockXHRInstance.onload?.();

    const result = await promise;
    expect(result.id).toBe("art-3");
  });

  it("rejects with ApiError when XHR returns non-201", async () => {
    const file = new File(["x"], "f.bin");

    const promise = uploadArtifact({
      token: "tok",
      workspaceId: "ws-1",
      file,
      artifactType: "log",
      onProgress: vi.fn(),
    });

    mockXHRInstance.status = 413;
    mockXHRInstance.responseText = JSON.stringify({
      error: { code: "artifact_too_large", message: "too big" },
    });
    mockXHRInstance.onload?.();

    await expect(promise).rejects.toThrow(ApiError);
  });
});

describe("downloadArtifact", () => {
  it("fetches signed URL and opens in new tab", async () => {
    const mockOpen = vi.fn();
    vi.stubGlobal("window", { open: mockOpen });

    mockFetch.mockResolvedValueOnce(
      jsonResponse({
        id: "art-1",
        url: "http://localhost:8080/v1/artifacts/art-1/content?expires=123&signature=abc",
        expires_at: "2026-04-13T00:05:00Z",
      }),
    );

    await downloadArtifact("my-token", "art-1");

    expect(mockFetch).toHaveBeenCalledWith(
      "http://localhost:8080/v1/artifacts/art-1/download",
      expect.objectContaining({
        headers: expect.objectContaining({
          Authorization: "Bearer my-token",
        }),
      }),
    );
    expect(mockOpen).toHaveBeenCalledWith(
      "http://localhost:8080/v1/artifacts/art-1/content?expires=123&signature=abc",
      "_blank",
    );
  });

  it("throws ApiError on 404", async () => {
    mockFetch.mockResolvedValueOnce(
      errorResponse(404, "artifact_not_found", "not found"),
    );

    await expect(downloadArtifact("tok", "bad-id")).rejects.toThrow(ApiError);
  });

  it("throws NetworkError when fetch fails", async () => {
    mockFetch.mockRejectedValueOnce(new TypeError("Failed to fetch"));

    await expect(downloadArtifact("tok", "art-1")).rejects.toThrow(
      NetworkError,
    );
  });
});
