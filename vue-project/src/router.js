
import { createRouter, createMemoryHistory, createWebHistory, createWebHashHistory } from 'vue-router'
import DashboardView from './views/DashboardView.vue'
import Host from './components/Host.vue'
import Device from './components/Device.vue'
import PageNotFound from './views/PageNotFound.vue'

const PRODUCTION = process.env.NODE_ENV === 'production'

export function newRouter(){
	const router = createRouter({
		history: createWebHashHistory(), //createWebHistory(import.meta.env.BASE_URL),
		routes: [
			{
				path: '/',
				name: 'dashboard',
				component: DashboardView,
				meta: {
					title: 'Dashboard',
				},
				children: [
					{
						path: '/:hostid',
						component: Host,
						props: true,
					},
					{
						path: '/:hostid/:deviceid',
						component: Device,
						props: true,
					},
				],
			},
			{
				path: '/admins',
				name: 'admin',
				component: () => import('./views/AdminView.vue'),
				meta: {
					title: 'Administrator',
				}
			},
			{
				path: "/:pathMatch(.*)*",
				component: PageNotFound,
				name: 'not-found',
				meta: {
					is404: true,
					title: '404 Not Found',
				}
			}
		],
		// scrollBehavior(to, from, savedPosition){
		// 	if(savedPosition){
		// 		return savedPosition
		// 	}
		// 	return
		// }
	})
	router.beforeEach((to, from) => {
		if(typeof document !== 'undefined'){
			var title = to.meta.title
			if(title || !from || from.path !== to.path){
				if(typeof title === 'function'){
					title = title(to)
				}
				if(title){
					title += ' - '
				}else{
					title = ''
				}
				document.title = title + 'CC Daemon'
			}
		}
	})
	return router
}
