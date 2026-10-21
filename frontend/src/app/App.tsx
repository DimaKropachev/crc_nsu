import { useEffect, type ReactNode } from 'react';
import { BrowserRouter, NavLink, Navigate, Outlet, Route, Routes, useLocation } from 'react-router-dom';
import { CalendarDays, FileText, GraduationCap, LayoutDashboard, LogOut, Mail, Sparkles, Users, BookOpenCheck, Menu, X } from 'lucide-react';
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
  { to: '/', label: 'Главная', icon: LayoutDashboard },
  { to: '/booking', label: 'Записаться', icon: CalendarDays },
  { to: '/consultations', label: 'Мои консультации', icon: BookOpenCheck },
  { to: '/resume', label: 'Анализ резюме', icon: FileText },
  { to: '/volunteer', label: 'Волонтёрство', icon: Users },
  { to: '/opportunities', label: 'Стипендии и стажировки', icon: GraduationCap },
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
  const [mobileOpen, setMobileOpen] = useState(false);
  const items = user?.role === 'admin' ? adminNav : studentNav;
  useEffect(() => setMobileOpen(false), [location.pathname]);
  const userName = `${user?.first_name ?? ''} ${user?.last_name ?? ''}`.trim();

  return <div className="app-shell">
    <aside className={`sidebar ${mobileOpen ? 'mobile-open' : ''}`}>
      <div className="sidebar-brand"><div className="brand-mark"><GraduationCap size={22} /></div><div><strong>Карьера НГУ</strong><span>Карьерный центр</span></div><button className="close-nav" aria-label="Закрыть меню" onClick={() => setMobileOpen(false)}><X size={18} /></button></div>
      <div className="nav-label">МЕНЮ</div>
      <nav className="side-nav">{items.map((item) => {
        const Icon = item.icon;
        return <NavLink key={item.to} to={item.to} end={item.to === '/'} className={({ isActive }) => `nav-link ${isActive ? 'active' : ''}`}><Icon size={19} strokeWidth={1.8} /><span>{item.label}</span>{item.to === '/resume' && <span className="nav-new">AI</span>}</NavLink>;
      })}</nav>
      <div className="sidebar-bottom"><div className="help-card"><span className="help-spark"><Sparkles size={17} /></span><strong>Есть карьерный вопрос?</strong><p>Подготовься к следующему шагу вместе с консультантом.</p><NavLink to={user?.role === 'admin' ? '/admin' : '/booking'}>Начать <span>↗</span></NavLink></div><button className="logout-link" onClick={logout}><LogOut size={18} /><span>{isMockMode ? 'Сменить демо-профиль' : 'Завершить сессию'}</span></button></div>
    </aside>
    {mobileOpen && <button className="nav-scrim" aria-label="Закрыть меню" onClick={() => setMobileOpen(false)} />}
    <div className="main-column">
      <header className="topbar"><div className="topbar-left"><button className="mobile-menu" aria-label="Открыть меню" onClick={() => setMobileOpen(true)}><Menu size={21} /></button><div className="breadcrumb">НГУ <span>/</span> <strong>{items.find((item) => item.to === location.pathname)?.label ?? 'Карьерный центр'}</strong></div></div><div className="topbar-profile"><div className="profile-copy"><strong>{userName}</strong><span>{user?.role === 'admin' ? 'Карьерный консультант' : 'Студент НГУ'}</span></div><Avatar name={userName} src={user?.avatar_url} /></div></header>
      {isMockMode && <div className="demo-banner"><span className="demo-dot" /><span>Демонстрационный режим — изменения не сохраняются на сервер</span></div>}
      <main className="main-content"><Outlet /></main>
      <footer className="app-footer"><span>© Карьерный центр НГУ</span><span>Сервис карьерного развития студентов</span></footer>
    </div>
    <nav className="bottom-nav">{items.slice(0, 5).map((item) => { const Icon = item.icon; return <NavLink key={item.to} to={item.to} end={item.to === '/'} className={({ isActive }) => `bottom-nav-item ${isActive ? 'active' : ''}`}><Icon size={20} /><span>{item.label.split(' ')[0]}</span></NavLink>; })}</nav>
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
