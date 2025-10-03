<script setup lang="ts">
import { ProfileViewDetailed } from "@atproto/api/dist/client/types/app/bsky/actor/defs";
import { getProfile } from "~/lib/cached-bsky";
import { Actor } from "../../../proto/bff/v1/types_pb";

const props = defineProps<{
  did: string;
  pending?: number;
  variant: "profile" | "queue";
}>();
const $emit = defineEmits<{
  (event: "next"): void;
}>();

const error = ref<string>();
const auditLog = ref() as Ref<{ refresh(): Promise<void> }>;

const subject = ref<ProfileViewDetailed>();
const actor = ref<Actor>();
async function loadProfile() {
  const api = await useAPI();
  const [profile, actr] = await Promise.all([
    getProfile(props.did),
    api.getActor({ did: props.did }),
  ]);
  subject.value = profile;
  actor.value = actr.actor;
}

async function refresh() {
  await loadProfile();
  await auditLog.value?.refresh();
}

function handleNext() {
  $emit("next");
  if (props.variant === "profile") {
    refresh();
  }
}

watch(
  () => props.did,
  () => refresh()
);

await refresh();
</script>

<template>
  <div>
    <user-card
      class="mb-5"
      :did="subject?.did || props.did"
      :actor="actor"
      :pending="pending"
      :variant="variant"
      @next="handleNext"
    />
    <shared-card v-if="error" variant="error">{{ error }}</shared-card>
    <user-audit-log
      v-else
      ref="auditLog"
      :subject="subject"
      :actor="actor"
      :did="subject?.did || props.did"
      @error="error = $event"
    />
  </div>
</template>
