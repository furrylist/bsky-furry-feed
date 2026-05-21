<script setup>
useHead({
  title: "furryli.st Admin",
  meta: [{ name: "robots", content: "noindex" }],
  link: [
    { rel: "preconnect", href: "https://fonts.googleapis.com" },
    { rel: "preconnect", href: "https://fonts.gstatic.com" },
    {
      href: "https://fonts.googleapis.com/css2?family=Inter:wght@400;700&display=swap",
      rel: "stylesheet",
    },
  ],
  bodyAttrs: {
    class: "bg-white dark:bg-gray-900 dark:text-white",
  },
});

const user = await useUser();
</script>

<template>
  <div v-if="user" class="max-w-[800px] py-4 px-3 mx-auto">
    <core-nav />
    <NuxtPage />
  </div>
  <core-login-page v-else />
</template>
