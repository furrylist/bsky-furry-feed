import { Timestamp } from "@bufbuild/protobuf";
import { AuditEvent } from "../proto/bff/v1/moderation_service_pb";

type AuditEventOrFollowedAt =
  | AuditEvent
  | {
      createdAt?: Timestamp;
      isFollowedAt: true;
      id: "follow";
      actorDid: string;
    };
