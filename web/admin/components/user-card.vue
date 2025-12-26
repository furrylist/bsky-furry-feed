<script lang="ts" setup>
import { ProfileViewDetailed } from "@atproto/api/dist/client/types/app/bsky/actor/defs";
import { Actor, ActorStatus } from "../../proto/bff/v1/types_pb";
import { getProfile } from "~/lib/cached-bsky";
import { newAgent } from "~/lib/auth";
import { addSISuffix } from "~/lib/util";
import { BlueskyLabel } from "~/composables/useBlueskyLabels";
import { AppBskyFeedDefs } from "@atproto/api";

const props = defineProps<{
  did: string;
  actor?: Actor;
  pending?: number;
  variant: "queue" | "profile";
}>();
const $emit = defineEmits(["next"]);

const currentActor = await useActor();
const showAvatarModal = ref(false);
const loading = ref(false);
const showRolesModal = ref(false);
const data = ref<ProfileViewDetailed>();
const labels = ref<Array<BlueskyLabel>>([]);
const loadProfile = async () => {
  const labelsQuery = useBlueskyLabels(props.did);
  data.value = await getProfile(props.did);

  posts.value = await newAgent()
    .getAuthorFeed({ actor: props.did })
    .then((r) => r.data.feed);
  labels.value = await labelsQuery;
};

async function next() {
  if (props.variant === "profile") {
    await loadProfile();
  }
  loading.value = false;
  $emit("next");
}

async function handleRoleUpdate() {
  showRolesModal.value = false;
  loading.value = true;
  await loadProfile();
  loading.value = false;
}

watch(
  () => props.did,
  () => loadProfile()
);

const posts = ref<AppBskyFeedDefs.FeedViewPost[]>([]);

await loadProfile();
</script>

<template>
  <template v-if="data">
    <user-queue-banner
      v-if="actor?.status === ActorStatus.PENDING"
      :did="data.did"
      :name="data.displayName || data.handle.replace(/.bsky.social$/, '')"
      :pending="pending"
      @next="next"
      @loading="loading = true"
    />
    <div
      class="flex max-md:flex-col gap-3"
      :class="{ 'loading-flash': loading }"
    >
      <div class="mb-3 md:w-[50%] card-list h-min flex-1">
        <user-actions :did="data.did" :status="actor?.status" @update="next" />
        <shared-card v-if="data.banner">
          <img
            :src="`https://bsky-cdn.codingpa.ws/banner/${did}/450x150`"
            class="w-full object-fit rounded-lg"
            height="101"
            width="304"
            alt=""
          />
        </shared-card>
        <shared-card>
          <div class="flex gap-3 items-center">
            <button
              class="relative flex overflow-hidden rounded-lg"
              @click="showAvatarModal = true"
            >
              <shared-avatar
                :did="data.did"
                :has-avatar="Boolean(data.avatar)"
                not-rounded
                resize="72x72"
                :size="72"
              />
              <span
                class="opacity-0 hover:opacity-100 transition duration-300 w-full h-full absolute flex items-center bg-black bg-opacity-50 text-xs uppercase tracking-tight"
              >
                Click to zoom
              </span>
            </button>
            <core-modal v-if="showAvatarModal" @close="showAvatarModal = false">
              <div class="z-10">
                <shared-avatar
                  class="w-auto h-auto max-h-[80vh] max-w-[80vw]"
                  :did="data.did"
                  :has-avatar="Boolean(data.avatar)"
                  resize="webp"
                  :size="512"
                />
              </div>
            </core-modal>
            <div class="flex flex-col">
              <div v-if="data.displayName" class="text-lg">
                {{ data.displayName }}
              </div>
              <div>
                <nuxt-link
                  class="underline hover:no-underline text-muted"
                  :href="`https://bsky.app/profile/${data.handle}`"
                  target="_blank"
                >
                  @{{ data.handle }}
                </nuxt-link>
              </div>
            </div>
          </div>
        </shared-card>
        <shared-card
          v-if="
            actor?.roles.length ||
            (actor && currentActor.isAdmin && variant !== 'queue')
          "
          class="flex items-center gap-1"
        >
          <icon-key class="text-muted" />
          <span
            v-for="role in actor.roles"
            :key="role"
            class="text-sm capitalize bg-gray-600 rounded-lg px-1 py-0.5 text-white"
          >
            {{ role }}
          </span>
          <span v-if="actor.roles.length === 0" class="text-muted"
            >No roles assigned.</span
          >
          <button
            v-if="currentActor.isAdmin"
            class="ml-auto text-sm rounded-lg py-0.5 px-1.5 border border-gray-300 dark:border-gray-700 hover:bg-zinc-700"
            :disabled="loading"
            @click="showRolesModal = true"
          >
            Edit
          </button>
          <user-role-edit-modal
            v-if="showRolesModal"
            :actor="actor"
            @cancel="showRolesModal = false"
            @update="handleRoleUpdate"
          />
        </shared-card>
        <shared-card class="meta">
          <div class="meta-item">
            <user-status-badge class="text-sm" :status="actor?.status" />
          </div>
          <span
            class="ml-auto meta-item"
            :title="`${data?.followersCount || 0} followers`"
          >
            <icon-users class="text-muted" />
            {{ addSISuffix(data?.followersCount) }}
          </span>
          <span class="meta-item" :title="`${data?.followsCount || 0} follows`">
            <icon-user-check class="text-muted" />
            {{ addSISuffix(data?.followsCount) }}
          </span>
          <span class="meta-item" :title="`${data?.postsCount || 0} posts`">
            <icon-square-bubble class="text-muted" :size="18" />
            {{ addSISuffix(data?.postsCount) }}
          </span>
        </shared-card>
        <shared-card v-if="labels.length > 0">
          <ul class="text-sm">
            <li
              v-for="label in labels"
              :key="`${label.src}:${label.val}`"
              class="flex items-center py-0.5 px-1 border border-gray-300 dark:border-gray-700 rounded-lg w-max"
            >
              <shared-avatar
                :did="label.src"
                :has-avatar="Boolean(label.labeler.avatar)"
                :size="20"
                resize="20x20"
                class="mr-1"
              />
              <div>
                <span class="text-muted text-xs"
                  >{{ label.labeler.handle }}/</span
                >{{ label.val }}
              </div>
            </li>
          </ul>
        </shared-card>
        <shared-card v-if="data.description">
          <shared-bsky-description :description="data.description" />
        </shared-card>
      </div>
      <div class="mb-3 md:w-[50%]">
        <shared-card
          v-if="actor"
          :class="{ 'loading-flash': loading }"
          no-padding
        >
          <user-recent-posts :actor-did="actor.did" :posts="posts" />
        </shared-card>
      </div>
    </div>
  </template>
  <div v-else>
    <user-queue-banner
      v-if="actor?.status === ActorStatus.PENDING"
      :did="props.did"
      :name="props.did"
      :pending="pending"
      reject-only
      @next="next"
      @loading="loading = true"
    />

    <shared-card class="bg-red-200 dark:bg-red-700">
      Profile with did {{ did }} was not found.
    </shared-card>
  </div>
</template>

<style scoped>
.card-list > :not(:last-of-type) {
  @apply border-b-0;
  @apply rounded-b-none;
}

.card-list > :not(:first-of-type) {
  @apply rounded-t-none;
}
</style>
