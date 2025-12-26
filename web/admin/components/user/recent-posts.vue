<script setup lang="ts">
import { addSISuffix } from "~/lib/util";
import { ViewImage } from "@atproto/api/dist/client/types/app/bsky/embed/images";
import { hasFurryHashtag } from "~/lib/furry-detector";
import { PostView } from "@atproto/api/dist/client/types/app/bsky/feed/defs";

const props = defineProps<{
  posts: Array<PostView>;
}>();

const hasFurryTags = computed(() =>
  props.posts.map((p) => String((p.record as any)?.text)).some(hasFurryHashtag)
);
</script>

<template>
  <div
    class="px-4 py-3 border-b border-gray-300 dark:border-gray-700 flex items-center"
  >
    <h2>Recent posts</h2>
    <span
      v-if="hasFurryTags"
      class="ml-auto text-sm bg-teal-700 flex items-center gap-0.5 px-1 rounded-lg h-min"
    >
      <icon-check />
      furry tags
    </span>
  </div>
  <div class="overflow-y-auto max-h-[500px]">
    <div
      v-for="post in posts"
      :key="post.uri"
      class="px-4 py-2 border-b border-gray-300 dark:border-gray-700"
    >
      <template v-if="post">
        <div class="meta text-sm text-muted">
          <span class="meta-item">
            <shared-date :date="new Date(post.indexedAt)" />
          </span>
          <span class="meta-item flex items-center gap-0.5">
            <icon-heart class="text-muted" />
            {{ addSISuffix(post.likeCount || 0) }}
          </span>
          <span class="meta-item flex items-center gap-0.5">
            <icon-square-bubble class="text-muted" :size="14" />
            {{ addSISuffix(post.replyCount || 0) }}
          </span>
        </div>
        <div class="flex">
          <shared-bsky-description
            :description="(post.record as any)?.text"
            class="flex-1"
          />
          <span
            v-if="post.embed && 'images' in post.embed"
            class="w-[25%] h-100 flex-shrink-0"
          >
            <img
              v-for="img in (post.embed.images as ViewImage[])"
              :key="img.thumb"
              class="object-cover h-100 rounded-lg"
              :src="img.thumb"
              :alt="img.alt"
            />
          </span>
        </div>
      </template>
      <div v-else class="text-sm text-muted">Error: post not found.</div>
    </div>
    <div
      v-if="posts.length === 0"
      class="text-muted px-4 py-2 border-gray-300 dark:border-gray-700"
    >
      No recent posts.
    </div>
  </div>
</template>
