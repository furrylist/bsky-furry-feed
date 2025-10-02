<script lang="ts" setup>
import { getProfile } from "~/lib/cached-bsky";
import { Actor, ActorStatus } from "../../proto/bff/v1/types_pb";
import { ProfileViewDetailed } from "@atproto/api/dist/client/types/app/bsky/actor/defs";
import { categorizeProfiles } from "~/lib/queues";
import { ApprovalQueueAction } from "../../proto/bff/v1/moderation_service_pb";
import { chunk } from "~/lib/util";

const api = await useAPI();

const actors = ref<Actor[]>([]);

const actorProfiles = ref<ProfileViewDetailed[]>([]);
const randomActor = ref<Actor>();
const currentQueue = ref<keyof typeof queues["value"]>("All");

const error = ref<string>();

const actorProfilesMap = computed(() => {
  const map = new Map<string, ProfileViewDetailed>();

  for (const profile of actorProfiles.value) {
    map.set(profile.did, profile);
  }

  return map;
});

const queues = computed(() =>
  categorizeProfiles(actors.value, actorProfilesMap.value)
);
const deletedAccountDIDs = computed(() =>
  queues.value.Empty.filter((p) => !didToProfile(p.did)).map((p) => p.did)
);
const rejectAllDeletedLoading = ref(false);

const nextActor = async () => {
  error.value = "";

  const queue = await api
    .listActors({ filterStatus: ActorStatus.PENDING })
    .catch((err) => {
      error.value = err.rawMessage;

      return {
        actors: [],
      };
    });

  actors.value = queue.actors;
  actorProfiles.value = await Promise.all(
    actors.value.map((a) => a.did).map(getProfile)
  ).then((p) => p.filter(Boolean));
  selectRandomActor();
};

await nextActor();

function didToProfile(did: string): ProfileViewDetailed | undefined {
  return actorProfilesMap.value.get(did);
}

function selectRandomActor() {
  if (!queues.value) return;
  const queue = queues.value[currentQueue.value];

  randomActor.value = queue[Math.floor(Math.random() * queue.length)] as Actor;
}

async function rejectAllDeleted() {
  rejectAllDeletedLoading.value = true;
  for (const dids of chunk(deletedAccountDIDs.value, 5)) {
    try {
      await Promise.all(
        dids.map(async (did) => {
          await api.processApprovalQueue({
            did,
            action: ApprovalQueueAction.REJECT,
            reason: "Profile no longer exists",
          });
        })
      );
    } catch (e) {
      error.value = `Failed rejecting deleted accounts: ${e}`;
      rejectAllDeletedLoading.value = false;
      await nextActor();
      return;
    }
  }
  rejectAllDeletedLoading.value = false;
  await nextActor();
}
</script>

<template>
  <div>
    <div class="flex gap-1 mb-3">
      <button
        v-for="(list, label) in queues"
        :key="label"
        class="border hover:bg-slate-800 border-gray-300 dark:border-gray-700 rounded-lg px-2 py-0.5 max-md:text-sm"
        :class="{
          'bg-slate-800': currentQueue === label,
          'ml-auto text-muted': label === 'Held back',
        }"
        @click="
          () => {
            currentQueue = label;
            selectRandomActor();
          }
        "
      >
        {{ label }} ({{ list.length }})
      </button>
    </div>

    <held-back-list
      v-if="currentQueue === 'Held back'"
      :users="
        queues['Held back'].map((actor) => ({ ...actor, ...didToProfile(actor.did) } as Actor & ProfileViewDetailed))
      "
    />
    <shared-card
      v-else-if="currentQueue === 'Empty' && deletedAccountDIDs.length > 5"
      class="flex flex-col py-2 gap-2 max-w-[400px] mx-auto"
    >
      <h2 class="text-lg font-bold">Many deleted accounts found</h2>
      <p>
        Sometimes users follow <code>@furryli.st</code> and later delete their
        account or get suspended. These accounts pile up in the queue and should
        be rejected in bulk.
      </p>
      <button
        class="py-0.5 max-md:py-1 max-md:px-3 px-2 mr-1 bg-red-500 dark:bg-red-600 hover:bg-red-600 dark:hover:bg-red-700 disabled:bg-red-400 disabled:dark:bg-red-500 rounded-lg disabled:cursor-not-allowed"
        :disabled="rejectAllDeletedLoading"
        @click="rejectAllDeleted"
      >
        Reject all {{ deletedAccountDIDs.length }} deleted accounts
      </button>
    </shared-card>
    <div v-else>
      <shared-card v-if="error" variant="error">{{ error }}</shared-card>
      <user-profile
        v-else-if="randomActor"
        :did="randomActor.did"
        :pending="queues[currentQueue].length"
        variant="queue"
        @next="nextActor"
      />
      <shared-card v-else>
        No user is in {{ currentQueue === "All" ? "the" : "this" }} queue.
      </shared-card>
    </div>
  </div>
</template>
