// Domänenspezifische Texte und Felder. app.js ist davon unabhängig.
const CFG = {
  appName: 'Fahrradverleih',
  item: 'Fahrrad',          // Singular
  oneItem: 'ein Fahrrad',
  items: 'Fahrräder',       // Plural
  itemsNav: 'Fahrrad-Verwaltung',
  apiPath: '/api/bikes',
  icon: '🚲',
  size: {
    field: 'frameSize',
    label: 'Rahmengröße',
    unit: '',
    input: { type: 'select', options: ['XS', 'S', 'M', 'L', 'XL', '16 Zoll', '20 Zoll', '24 Zoll'], default: 'M' },
    format: (v) => v,
  },
};
