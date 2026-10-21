import { useEffect, type ReactNode } from 'react';
import { BrowserRouter, NavLink, Navigate, Outlet, Route, Routes, useLocation, useNavigate } from 'react-router-dom';
import { Bell, BriefcaseBusiness, CalendarDays, ChevronRight, FileText, GraduationCap, HeartHandshake, Home, LayoutDashboard, LogOut, Mail, Sparkles, BookOpenCheck, Menu, ShieldCheck } from 'lucide-react';
import { useAuth } from './AuthProvider';
import { isMockMode } from '../shared/api/api';
import { Avatar, Button, LoadingState, Notice } from '../shared/ui';
import { DashboardPage } from '../pages/DashboardPage';
import { BookingPage } from '../pages/BookingPage';
import { ConsultationsPage } from '../pages/ConsultationsPage';
import { ResumeAnalysisPage } from '../pages/ResumeAnalysisPage';
import { AdminPage } from '../pages/AdminPage';
import { StubPage } from '../pages/StubPage';
import { useState } from 'react';

interface NavItem { to: string; label: string; icon: typeof LayoutDashboard; roles?: Array<'student' | 'admin'>; }
const studentNav: NavItem[] = [
  { to: '/', label: 'Главная', icon: Home },
  { to: '/booking', label: 'Запись на консультацию', icon: CalendarDays },
  { to: '/consultations', label: 'Мои консультации', icon: BookOpenCheck },
  { to: '/resume', label: 'Анализ резюме', icon: FileText },
  { to: '/opportunities', label: 'Стажировки и стипендии', icon: BriefcaseBusiness },
  { to: '/volunteer', label: 'Волонтёрство', icon: HeartHandshake },
];
const adminNav: NavItem[] = [
  { to: '/', label: 'Обзор', icon: LayoutDashboard },
  { to: '/admin', label: 'Консультации и расписание', icon: CalendarDays },
  { to: '/mailings', label: 'Рассылки', icon: Mail },
];

function AuthScreen() {
  const { loading, error, loginDemo } = useAuth();
  if (loading) return <main className="auth-screen"><LoadingState label="Входим в сервис НГУ…" /></main>;
  return <main className="auth-screen"><div className="auth-card">
    <div className="brand-mark large"><GraduationCap size={28} /></div>
    <div className="eyebrow">НГУ · КАРЬЕРНЫЙ ЦЕНТР</div>
    <h1>Твоя карьера<br />начинается здесь</h1>
    <p className="auth-copy">Запишись на консультацию, подготовь резюме и наметь следующие шаги в карьере вместе с экспертами.</p>
    {error && <Notice tone="warning">{error}</Notice>}
    {error && !isMockMode && <Button variant="secondary" className="full-width" onClick={() => window.location.reload()}>Повторить вход</Button>}
    {isMockMode ? <div className="demo-login"><p className="small-muted">Локальный режим разработки. Данные демонстрационные.</p><Button className="full-width" onClick={() => loginDemo('student')}>Продолжить как студент</Button><Button className="full-width" variant="secondary" onClick={() => loginDemo('admin')}>Открыть кабинет консультанта</Button></div> : <div className="auth-hint"><span className="vk-mark">vk</span><div><strong>Автоматический вход через VK</strong><p>Откройте приложение из VK. Дополнительный логин или пароль не нужны.</p></div></div>}
    <div className="auth-footnote">Карьерный центр Новосибирского государственного университета</div>
  </div></main>;
}

function AppShell() {
  const { user, logout } = useAuth();
  const location = useLocation();
  const navigate = useNavigate();
  const [mobileOpen, setMobileOpen] = useState(false);
  const items = user?.role === 'admin' ? adminNav : studentNav;
  useEffect(() => setMobileOpen(false), [location.pathname]);
  const userName = `${user?.first_name ?? ''} ${user?.last_name ?? ''}`.trim();

  const pageTitle = location.pathname.startsWith('/admin') ? 'Кабинет консультанта' : location.pathname === '/booking' ? 'Карьерная консультация' : location.pathname === '/resume' ? 'Помощник по резюме' : 'Пространство для твоей карьеры';

  return <div className="app-shell">
    <aside className={`sidebar ${mobileOpen ? 'sidebar-open' : ''}`}>
      <NavLink className="brand" to="/" aria-label="На главную"><img src={`${import.meta.env.BASE_URL}NSU_logo.png`} alt="Логотип НГУ" /></NavLink>
      <div className="nav-caption">РАБОЧЕЕ ПРОСТРАНСТВО</div>
      <nav className="main-nav" aria-label="Основная навигация">{items.map((item) => {
        const Icon = item.icon;
        return <NavLink key={item.to} to={item.to} end={item.to === '/'} className={({ isActive }) => `nav-link ${isActive ? 'nav-link-active' : ''}`}><Icon size={19} strokeWidth={1.8} /><span>{item.label}</span>{item.to === '/resume' && <span className="nav-badge">AI</span>}<ChevronRight size={15} className="nav-chevron" /></NavLink>;
      })}</nav>
      <div className="sidebar-spacer" /><div className="sidebar-help"><div className="help-icon"><Sparkles size={18} /></div><b>Следующий шаг — за тобой</b><p>Маленькие действия складываются в большую карьеру.</p><button type="button" onClick={() => navigate(user?.role === 'admin' ? '/admin' : '/opportunities')}>{user?.role === 'admin' ? 'К расписанию' : 'Изучить возможности'} <ChevronRight size={14} /></button></div>
      <div className="sidebar-user"><Avatar name={userName} src={user?.avatar_url} size="sm" /><div className="sidebar-user-copy"><b>{userName}</b><small>{user?.role === 'admin' ? 'Консультант' : 'Студент НГУ'}</small></div><button className="icon-button user-exit" type="button" onClick={logout} aria-label={isMockMode ? 'Сменить демо-профиль' : 'Завершить сессию'}><LogOut size={17} /></button></div>
    </aside>
    {mobileOpen && <button className="mobile-scrim" aria-label="Закрыть меню" onClick={() => setMobileOpen(false)} />}
    <div className="main-area">
      <header className="topbar"><div className="topbar-left"><button className="icon-button mobile-menu-button" aria-label="Открыть меню" onClick={() => setMobileOpen(true)}><Menu size={21} /></button><div><div className="topbar-kicker">НГУ · ЦЕНТР РАЗВИТИЯ КАРЬЕРЫ</div><div className="topbar-title">{pageTitle}</div></div></div><div className="topbar-actions">{isMockMode && <span className="demo-pill"><i />Демо-режим</span>}<button className="icon-button notification-button" type="button" onClick={() => navigate(user?.role === 'admin' ? '/admin' : '/consultations')} aria-label="Мои записи"><Bell size={18} /></button><div className="topbar-profile"><Avatar name={userName} src={user?.avatar_url} size="sm" /><span>{user?.first_name || 'Пользователь'}</span></div></div></header>
      {isMockMode && <div className="demo-banner"><ShieldCheck size={15} /><span>Локальные демонстрационные данные. Операции не попадают в реальную систему НГУ.</span></div>}
      <main className="page-content"><Outlet /></main>
      <footer className="app-footer"><span>Карьерный центр НГУ</span><span>Создано для твоего профессионального старта</span></footer>
    </div>
    <nav className="mobile-bottom-nav" aria-label="Быстрая навигация">{items.slice(0, 4).map((item) => { const Icon = item.icon; return <NavLink key={item.to} to={item.to} end={item.to === '/'} className={({ isActive }) => `mobile-nav-item ${isActive ? 'mobile-nav-active' : ''}`}><Icon size={19} /><span>{item.to === '/booking' ? 'Запись' : item.to === '/consultations' ? 'Мои встречи' : item.to === '/resume' ? 'Резюме' : item.label.split(' ')[0]}</span></NavLink>; })}</nav>
  </div>;
}

function RoleGuard({ role, children }: { role: 'student' | 'admin'; children: ReactNode }) {
  const { user } = useAuth();
  if (!user) return <Navigate to="/" replace />;
  if (user.role !== role) return <Navigate to="/" replace />;
  return children;
}

export function App() {
  const { user, loading } = useAuth();
  if (loading) return <main className="auth-screen"><LoadingState /></main>;
  if (!user) return <AuthScreen />;
  return <BrowserRouter><Routes><Route element={<AppShell />}>
    <Route path="/" element={<DashboardPage />} />
    <Route path="/booking" element={<RoleGuard role="student"><BookingPage /></RoleGuard>} />
    <Route path="/consultations" element={<RoleGuard role="student"><ConsultationsPage /></RoleGuard>} />
    <Route path="/resume" element={<RoleGuard role="student"><ResumeAnalysisPage /></RoleGuard>} />
    <Route path="/admin" element={<RoleGuard role="admin"><AdminPage /></RoleGuard>} />
    <Route path="/volunteer" element={<RoleGuard role="student"><StubPage kind="volunteer" /></RoleGuard>} />
    <Route path="/opportunities" element={<RoleGuard role="student"><StubPage kind="opportunities" /></RoleGuard>} />
    <Route path="/mailings" element={<RoleGuard role="admin"><StubPage kind="mailings" /></RoleGuard>} />
    <Route path="*" element={<Navigate to="/" replace />} />
  </Route></Routes></BrowserRouter>;
}
