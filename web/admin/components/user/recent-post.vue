<script setup lang="ts">
import { addSISuffix } from "~/lib/util";
import { ViewImage } from "@atproto/api/dist/client/types/app/bsky/embed/images";

import { AppBskyFeedDefs } from "@atproto/api";
import { ActorStatus } from "../../../proto/bff/v1/types_pb";

const props = defineProps<{
  actorDid: string;
  post: AppBskyFeedDefs.FeedViewPost;
}>();

const postType = computed(() => {
  if (props.post.reply) return "reply";
  if (props.post.reason?.$type === "app.bsky.feed.defs#reasonRepost")
    return props.post.post.author.did === props.actorDid
      ? "self-repost"
      : "repost";
  if (props.post.post.author.did !== props.actorDid) return "unknown-other";
  return "post";
});
const showPost = computed(
  () =>
    postType.value === "post" ||
    postType.value === "repost" ||
    postType.value === "self-repost"
);

const authorStatus = ref<ActorStatus>();

onMounted(async () => {
  if (postType.value === "repost") {
    const api = await useAPI();
    const resp = await api.getActor({ did: props.post.post.author.did });
    authorStatus.value = resp.actor?.status;
  }
});
</script>

<template>
  <div
    v-if="showPost"
    class="px-4 py-2 border-b border-gray-300 dark:border-gray-700"
    :class="postType === 'repost' ? 'bg-gray-200/40 dark:bg-gray-950/40' : ''"
  >
    <div
      v-if="postType === 'repost' || postType === 'self-repost'"
      class="text-muted opacity-80 w-full flex items-center gap-0.5 mb-0.5 text-xs"
    >
      <icon-reskeet class="h-4 w-4" />
      <span v-if="postType === 'self-repost'">Self-reposting</span>
      <span v-else
        >Reposting
        <nuxt-link
          class="underline hover:no-underline"
          :href="`/users/${post.post.author.did}`"
          >@{{ post.post.author.handle }}</nuxt-link
        >
        <user-status-badge
          v-if="authorStatus"
          class="ml-1"
          :status="authorStatus"
          tiny
        />
      </span>
    </div>
    <div class="meta text-sm text-muted">
      <span class="meta-item">
        <shared-date :date="new Date(post.post.indexedAt)" />
      </span>
      <span class="meta-item flex items-center gap-0.5">
        <icon-heart class="text-muted" />
        {{ addSISuffix(post.post.likeCount || 0) }}
      </span>
      <span class="meta-item flex items-center gap-0.5">
        <icon-square-bubble class="text-muted" :size="14" />
        {{ addSISuffix(post.post.replyCount || 0) }}
      </span>
    </div>
    <div class="flex">
      <shared-bsky-description
        :description="(post.post.record as any)?.text"
        class="flex-1"
      />
      <span
        v-if="post.post.embed && 'images' in post.post.embed"
        class="w-[25%] h-100 flex-shrink-0"
      >
        <img
          v-for="img in (post.post.embed.images as ViewImage[])"
          :key="img.thumb"
          class="object-cover h-100 rounded-lg"
          :src="img.thumb"
          :alt="img.alt"
        />
      </span>
    </div>
  </div>
</template>
