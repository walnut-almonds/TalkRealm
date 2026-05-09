import { createRouter, createWebHashHistory } from 'vue-router'

// Lazy-loaded views
const ChatView = () => import('@/views/ChatView.vue')
const FeedView = () => import('@/views/FeedView.vue')
const LearnView = () => import('@/views/LearnView.vue')
const LiveView = () => import('@/views/LiveView.vue')

export const routes = [
    { path: '/', name: 'chat', component: ChatView, meta: { section: 'chat' } },
    { path: '/feed', name: 'feed', component: FeedView, meta: { section: 'feed' } },
    { path: '/learn', name: 'learn', component: LearnView, meta: { section: 'learn' } },
    { path: '/live', name: 'live', component: LiveView, meta: { section: 'live' } },
    // Fallback
    { path: '/:pathMatch(.*)*', redirect: '/' },
]

const router = createRouter({
    history: createWebHashHistory(),
    routes,
})

export default router
