import { mount } from 'svelte'
import App from '@/App.svelte'
import '@/app.css'

const app = document.getElementById('app')
if (app) mount(App, { target: app })
