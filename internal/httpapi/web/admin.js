import {icon,escapeHTML} from './ui.js';
import {renderDashboard,renderBookings,renderCalendar} from './admin-overview.js';
import {renderResources,renderServices,renderActivity,renderSettings} from './admin-config.js';
export {openCreateBooking} from './admin-booking.js';

export const navItems=[
 {id:'dashboard',label:'Обзор',icon:'dashboard'}, {id:'calendar',label:'Календарь',icon:'calendar'},
 {id:'bookings',label:'Бронирования',icon:'bookings',badge:true}, {id:'resources',label:'Ресурсы',icon:'resources'},
 {id:'services',label:'Услуги',icon:'sparkles'}, {id:'activity',label:'История действий',icon:'activity'},
 {id:'settings',label:'Настройки',icon:'settings'}
];
export const pageMeta={dashboard:['Обзор','Операционный центр'],calendar:['Календарь','Ресурсы и загрузка'],bookings:['Бронирования','Все каналы в одном месте'],resources:['Ресурсы','Пространства и оборудование'],services:['Услуги','Время, цены и подтверждения'],activity:['История действий','Неизменяемый журнал'],settings:['Настройки','Организация и публичная запись']};
const pages={dashboard:renderDashboard,calendar:renderCalendar,bookings:renderBookings,resources:renderResources,services:renderServices,activity:renderActivity,settings:renderSettings};
export async function renderAdminPage(page,root,ctx){try{await (pages[page]||pages.dashboard)(root,ctx)}catch(e){root.innerHTML=`<div class="fatal in-page">${icon('warning')}<h2>Не удалось загрузить раздел</h2><p>${escapeHTML(e.message)}</p><button class="btn secondary" id="retry">${icon('refresh')} Повторить</button></div>`;root.querySelector('#retry').onclick=()=>renderAdminPage(page,root,ctx)}}
