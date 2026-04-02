import { redirect } from '@sveltejs/kit';
import type { PageLoad } from './$types';

export const load = (async ({ parent, params }) => {
  const { feeds } = await parent();
  const feed = feeds?.find((f) => f.id === params.feed);

  if (!feed) {
    throw redirect(302, '/welcome');
  }

  return { feed };
}) satisfies PageLoad;
