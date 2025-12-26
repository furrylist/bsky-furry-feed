<script setup lang="ts">
import { addSISuffix } from "~/lib/util";
import { ViewImage } from "@atproto/api/dist/client/types/app/bsky/embed/images";

import { AppBskyFeedDefs } from "@atproto/api";

const props = defineProps<{
  actorDid: string;
  post: AppBskyFeedDefs.FeedViewPost;
}>();

const postType = computed(() => {
  if (props.post.reply) return "reply";
  if (props.post.reason?.$type === "app.bsky.feed.defs#reasonRepost")
    return "repost";
  if (props.post.post.author.did !== props.actorDid) return "unknown-other";
  return "post";
});
const showPost = computed(() => postType.value === "post");
</script>

<template>
  <div
    v-if="showPost"
    class="px-4 py-2 border-b border-gray-300 dark:border-gray-700"
  >
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
