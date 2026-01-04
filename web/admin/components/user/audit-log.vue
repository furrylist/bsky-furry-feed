<script setup lang="ts">
import { ProfileViewDetailed } from "@atproto/api/dist/client/types/app/bsky/actor/defs";
import { AuditEvent } from "../../../proto/bff/v1/moderation_service_pb";
import { Actor } from "../../../proto/bff/v1/types_pb";
import { AuditEventOrFollowedAt } from "~/types";

const $emit = defineEmits<{
  (event: "error", message: string): void;
}>();
const props = defineProps<{
  did: string;
  subject?: ProfileViewDetailed;
  actor?: Actor;
}>();
const loading = ref(true);

const auditEvents: Ref<AuditEvent[]> = ref([]);
const allAuditEvents: Ref<AuditEventOrFollowedAt[]> = computed(() => {
  const arr: AuditEventOrFollowedAt[] = [...auditEvents.value];
  if (
    props.actor &&
    !auditEvents.value.some((a) =>
      a.payload?.typeUrl.includes("bff.v1.CreateActorAuditPayload")
    )
  ) {
    if (props.actor.createdAt?.toDate()) {
      arr.unshift({
        actorDid: props.did,
        isFollowedAt: true,
        createdAt: props.actor.createdAt,
        id: "follow",
      });
    }
  }
  return arr;
});

async function loadEvents() {
  loading.value = true;
  $emit("error", "");

  const response = await api
    .listAuditEvents({
      filterSubjectDid: props.did,
    })
    .catch((err) => {
      $emit("error", err.rawMessage);
      return {
        auditEvents: [],
      };
    });
  auditEvents.value = response.auditEvents;
  loading.value = false;
}

defineExpose({
  refresh() {
    loadEvents();
  },
});

async function comment(comment: string) {
  $emit("error", "");

  let ok = true;

  await api
    .createCommentAuditEvent({
      subjectDid: props.did,
      comment,
    })
    .catch((err) => {
      ok = false;
      $emit("error", err.rawMessage);
    });

  if (ok) await loadEvents();
}
const api = await useAPI();
onMounted(async () => {
  await loadEvents();
  2;
});
</script>

<template>
  <h2 class="font-bold mb-3">Comments</h2>
  <template v-if="loading">
    <div
      v-for="delay in [100, 200, 300]"
      :key="delay"
      class="flex items-center my-4 loading-flash"
      :style="{ animationDelay: `${delay}ms` }"
    >
      <div
        class="bg-gray-200 dark:bg-gray-800 rounded-full flex items-center justify-center h-7 w-7 mr-3 flex-shrink-0"
      >
        &nbsp;
      </div>
      <div class="flex-1">
        <div
          class="flex max-md:flex-wrap items-center gap-1 rounded-lg bg-gray-200 dark:bg-gray-800"
        >
          &nbsp;
        </div>
      </div>
    </div>
  </template>
  <template v-else>
    <action
      v-for="action in allAuditEvents.sort(
        (a, b) =>
          (a.createdAt?.toDate().getTime() || 0) -
          (b.createdAt?.toDate().getTime() || 0)
      )"
      :key="action.id"
      :action="action"
    />
    <p v-if="allAuditEvents.length === 0" class="text-muted">
      No comments or audit events.
    </p>
    <shared-comment-box v-if="subject" @comment="comment" />
  </template>
</template>
