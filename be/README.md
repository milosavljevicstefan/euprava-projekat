E-Uprava Projekat: OpenPreschool
Kako pokrenuti projekt
1. Instalirajte Docker i Docker Compose.

2. U root folderu projekta pokrenite: `docker compose up --build`

3. Servisi će biti dostupni na:

  • Preschool Service: `http://localhost:8081`

  • Open Data Service: `http://localhost:8082`

  • Auth SSO: `http://localhost:8083`

Pravila rada (Git Workflow)
• Svaki novi feature mora imati svoju granu (npr. `feature/naziv-funkcionalnosti`).

• Zabranjeno je direktno push-anje na main granu.

• Pull Request (Merge): Nema spajanja grana u `main` dok drugi član tima ne pregleda i ne odobri kod.

• Prije svakog rada povucite najnovije promjene sa `main` grane.
