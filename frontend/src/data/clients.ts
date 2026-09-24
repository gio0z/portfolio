export interface Client {
  name: string;
  sector: string;
}

export const clients: Client[] = [
  { name: 'Pemerintah Kabupaten Blitar', sector: 'Government' },
  { name: 'Labtu', sector: 'Government' },
  { name: 'BA-Rekon', sector: 'Asset Reconciliation' },
  { name: 'CS Portal', sector: 'Customer Service' },
  { name: 'Afsa Tour & Transport', sector: 'Travel' },
  { name: 'Plantea', sector: 'Botanical Export' },
  { name: 'MD Fashion', sector: 'Commerce' },
  { name: 'Pamonta', sector: 'Asset Inventory' },
];
