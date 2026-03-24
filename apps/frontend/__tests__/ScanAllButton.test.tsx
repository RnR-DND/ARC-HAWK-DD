import React from 'react';
import { render, screen } from '@testing-library/react';
import ScanAllButton from '../components/ScanAllButton';

test('renders scan all button', () => {
  render(<ScanAllButton />);
  const buttonElement = screen.getByText(/Scan All Connected Assets/i);
  expect(buttonElement).toBeInTheDocument();
});