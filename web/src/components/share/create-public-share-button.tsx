"use client";

import { useEffect, useState } from "react";
import { useAccessToken } from "@workos-inc/authkit-nextjs/components";
import { Copy, EyeOff, Globe2, Loader2, Share2 } from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { createApiClient } from "@/lib/api/client";
import type {
  CreatePublicShareLinkResponse,
  PublicShareResourceType,
} from "@/lib/api/types";

interface CreatePublicShareButtonProps {
  resourceType: PublicShareResourceType;
  resourceId: string;
  label?: string;
  size?: "xs" | "sm" | "default";
  variant?: "outline" | "ghost" | "secondary" | "default";
  disabled?: boolean;
}

export function CreatePublicShareButton({
  resourceType,
  resourceId,
  label = "Share",
  size = "sm",
  variant = "outline",
  disabled,
}: CreatePublicShareButtonProps) {
  const { getAccessToken } = useAccessToken();
  const [sharing, setSharing] = useState<"private" | "publication" | "unpublish" | null>(null);
  const [privateURL, setPrivateURL] = useState<string | null>(null);
  const [publicationURL, setPublicationURL] = useState<string | null>(null);
  const [shareID, setShareID] = useState<string | null>(null);

  useEffect(() => {
    setPrivateURL(null);
    setPublicationURL(null);
    setShareID(null);
  }, [resourceType, resourceId]);

  async function handleShare(searchIndexing: boolean) {
    const mode = searchIndexing ? "publication" : "private";
    const existingURL = searchIndexing ? publicationURL : privateURL;
    if (existingURL) {
      await copyURL(existingURL, searchIndexing);
      return;
    }

    setSharing(mode);
    try {
      const token = await getAccessToken();
      const api = createApiClient(token);
      const result = await api.post<CreatePublicShareLinkResponse>(
        "/v1/share-links",
        {
          resource_type: resourceType,
          resource_id: resourceId,
          search_indexing: searchIndexing,
        },
      );
      setShareID(result.share.id);
      setPrivateURL(result.url);
      if (searchIndexing && result.publication_url) {
        setPublicationURL(result.publication_url);
        void notifyPublicationChange(result.share.id);
        await copyURL(result.publication_url, true);
      } else {
        await copyURL(result.url, false);
      }
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to create share link",
      );
    } finally {
      setSharing(null);
    }
  }

  async function handleUnpublish() {
    if (!shareID) return;
    setSharing("unpublish");
    try {
      const token = await getAccessToken();
      await createApiClient(token).patch(`/v1/share-links/${shareID}`, {
        search_indexing: false,
      });
      setPublicationURL(null);
      void notifyPublicationChange(shareID);
      toast.success("Publication removed from the public index");
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to unpublish artifact");
    } finally {
      setSharing(null);
    }
  }

  return (
    <span className="inline-flex items-center gap-1.5">
      <Button
        type="button"
        variant={variant}
        size={size}
        onClick={() => void handleShare(false)}
        disabled={disabled || sharing !== null}
        title={privateURL ? "Copy private capability link" : "Create private capability link"}
      >
        {sharing === "private" ? (
          <Loader2 className="size-3.5 animate-spin" />
        ) : privateURL ? (
          <Copy className="size-3.5" />
        ) : (
          <Share2 className="size-3.5" />
        )}
        {privateURL ? "Copy link" : label}
      </Button>
      <Button
        type="button"
        variant="ghost"
        size={size}
        onClick={() => void handleShare(true)}
        disabled={disabled || sharing !== null}
        title="Publish a redacted, indexable artifact"
      >
        {sharing === "publication" ? (
          <Loader2 className="size-3.5 animate-spin" />
        ) : publicationURL ? (
          <Copy className="size-3.5" />
        ) : (
          <Globe2 className="size-3.5" />
        )}
        {publicationURL ? "Copy publication" : "Publish"}
      </Button>
      {publicationURL ? (
        <Button
          type="button"
          variant="ghost"
          size={size}
          onClick={() => void handleUnpublish()}
          disabled={disabled || sharing !== null}
          title="Remove this artifact from the public index"
        >
          {sharing === "unpublish" ? (
            <Loader2 className="size-3.5 animate-spin" />
          ) : (
            <EyeOff className="size-3.5" />
          )}
          Unpublish
        </Button>
      ) : null}
    </span>
  );
}

async function notifyPublicationChange(publicationID: string) {
  try {
    await fetch("/api/indexnow/publication", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ publication_id: publicationID }),
    });
  } catch {
    // Best-effort discovery hint. Publication state is already committed.
  }
}

async function copyURL(url: string, publication: boolean) {
  try {
    await navigator.clipboard.writeText(url);
    toast.success(publication ? "Publication link copied" : "Private share link copied");
  } catch {
    toast.success(publication ? "Publication created" : "Private share created", {
      description: url,
    });
  }
}
