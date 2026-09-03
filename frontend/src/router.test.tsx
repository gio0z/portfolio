import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { AppRouter } from './router';

describe('AppRouter', () => {
  beforeEach(() => {
    window.history.pushState({}, '', '/');
  });

  it('renders the admin shell only for /admin routes', () => {
    window.history.pushState({}, '', '/admin/reviews');
    render(<AppRouter />);
    expect(screen.getByRole('heading', { name: /review queue/i })).toBeInTheDocument();
    expect(screen.queryByText(/featured engineering work/i)).not.toBeInTheDocument();
  });

  it('renders the admin shell for /admin base route', () => {
    window.history.pushState({}, '', '/admin');
    render(<AppRouter />);
    expect(screen.getByRole('heading', { name: /review queue/i })).toBeInTheDocument();
    expect(screen.queryByText(/featured engineering work/i)).not.toBeInTheDocument();
  });

  it('renders the public portfolio shell for / routes', () => {
    window.history.pushState({}, '', '/');
    render(<AppRouter />);
    expect(screen.getByRole('heading', { name: /featured engineering work/i })).toBeInTheDocument();
    expect(screen.queryByRole('heading', { name: /review queue/i })).not.toBeInTheDocument();
  });

  it('renders the lab shell for /lab routes', () => {
    window.history.pushState({}, '', '/lab');
    render(<AppRouter />);
    expect(screen.getByRole('heading', { name: /design lab/i })).toBeInTheDocument();
    expect(screen.queryByText(/featured engineering work/i)).not.toBeInTheDocument();
  });

  it('renders the lab shell for /lab/* nested routes', () => {
    window.history.pushState({}, '', '/lab/checkout-redesign');
    render(<AppRouter />);
    expect(screen.getByRole('heading', { name: /design lab/i })).toBeInTheDocument();
    expect(screen.queryByText(/featured engineering work/i)).not.toBeInTheDocument();
  });
});
