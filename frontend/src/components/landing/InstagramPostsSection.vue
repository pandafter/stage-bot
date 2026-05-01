<script setup lang="ts">
import { ref, onMounted } from "vue";

interface Post {
  id: string;
  media_url: string;
  caption: string;
  permalink: string;
}

const posts = ref<Post[]>([]);
const loading = ref(true);

onMounted(async () => {
  try {
    const res = await fetch("/api/instagram");
    posts.value = await res.json();
  } catch (e) {
    console.error(e);
  } finally {
    loading.value = false;
  }
});
</script>

<template>
  <section class="section publicaciones" id="publicaciones">
    <div class="container">
      <span class="eyebrow">Instagram</span>
      <h2>Nuestras últimas<br /><span class="gold">publicaciones</span></h2>

      <div v-if="loading" class="grid">
        <div v-for="i in 6" :key="i" class="skeleton"></div>
      </div>

      <div v-else class="grid">
        <a
          v-for="post in posts"
          :key="post.id"
          :href="post.permalink"
          target="_blank"
          class="card"
        >
          <img :src="post.media_url" loading="lazy" />
          <div class="overlay">
            <p>{{ post.caption }}</p>
          </div>
        </a>
      </div>

      <div class="cta">
        <a
          href="https://instagram.com/scuderia_stage4"
          target="_blank"
          class="btn"
        >
          Ver perfil completo
        </a>
      </div>
    </div>
  </section>
</template>

<style scoped>
.publicaciones {
  background: #060606;
  color: white;
}

.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 16px;
  margin-top: 2rem;
}

/* CARD */
.card {
  position: relative;
  border-radius: 14px;
  overflow: hidden;
  aspect-ratio: 1;
}

.card img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  transition: 0.4s ease;
}

.card:hover img {
  transform: scale(1.08);
}

/* OVERLAY */
.overlay {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: flex-end;
  padding: 1rem;
  background: linear-gradient(transparent, rgba(0,0,0,0.9));
  opacity: 0;
  transition: 0.3s;
}

.card:hover .overlay {
  opacity: 1;
}

/* SKELETON */
.skeleton {
  background: #111;
  border-radius: 14px;
  aspect-ratio: 1;
  animation: pulse 1.2s infinite;
}

@keyframes pulse {
  0% { opacity: 0.6; }
  50% { opacity: 1; }
  100% { opacity: 0.6; }
}

/* CTA */
.cta {
  text-align: center;
  margin-top: 2rem;
}

.btn {
  background: #d4af37;
  padding: 0.8rem 1.5rem;
  border-radius: 8px;
  color: black;
  text-decoration: none;
}
</style>