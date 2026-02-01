// One-off command to create tabs Resumo, Transacoes, Ativos, Historico and move data.
//
// Run from repo root (after auth):
//   go run ./cmd/sheets-move-tabs -spreadsheet-id 1uXk6YoWhzEa8kfFujeLUmDJac6gNSwGuHjKgym_goFo
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"

	"github.com/matheusbuniotto/go-google-mcp/pkg/auth"
	sheetssvc "github.com/matheusbuniotto/go-google-mcp/pkg/services/sheets"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/sheets/v4"
)

func main() {
	spreadsheetID := flag.String("spreadsheet-id", "", "Spreadsheet ID")
	creds := flag.String("creds", "", "Path to credentials JSON (optional)")
	flag.Parse()
	if *spreadsheetID == "" {
		log.Fatal(" -spreadsheet-id is required")
	}

	scopes := []string{drive.DriveScope, sheets.SpreadsheetsScope}
	opts, err := auth.GetClientOptions(context.Background(), *creds, scopes)
	if err != nil {
		log.Fatalf("Auth: %v", err)
	}

	svc, err := sheetssvc.New(context.Background(), opts...)
	if err != nil {
		log.Fatalf("Sheets service: %v", err)
	}

	sp, err := svc.GetSpreadsheet(*spreadsheetID)
	if err != nil {
		log.Fatalf("Get spreadsheet: %v", err)
	}
	if len(sp.Sheets) == 0 {
		log.Fatal("Spreadsheet has no sheets")
	}
	firstSheetID := sp.Sheets[0].Properties.SheetId

	// Batch: rename first sheet to Resumo, add Transacoes, Ativos, Historico
	req := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				UpdateSheetProperties: &sheets.UpdateSheetPropertiesRequest{
					Properties: &sheets.SheetProperties{SheetId: firstSheetID, Title: "Resumo"},
					Fields:     "title",
				},
			},
			{AddSheet: &sheets.AddSheetRequest{Properties: &sheets.SheetProperties{Title: "Transacoes"}}},
			{AddSheet: &sheets.AddSheetRequest{Properties: &sheets.SheetProperties{Title: "Ativos"}}},
			{AddSheet: &sheets.AddSheetRequest{Properties: &sheets.SheetProperties{Title: "Historico"}}},
		},
	}
	_, err = svc.BatchUpdate(*spreadsheetID, req)
	if err != nil {
		log.Fatalf("Batch update: %v", err)
	}
	fmt.Println("Created tabs: Resumo (renamed), Transacoes, Ativos, Historico")

	// Resumo: headers + 3 data rows + total row (no section titles)
	resumoJSON := mustMarshal([][]interface{}{
		{"Categoria", "Ativo/Ticker", "Quantidade", "Preco Atual", "Valor Total (BRL)", "Alocacao %", "Yield Esperado", "Data Ultima Atualizacao"},
		{"Tesouro IPCA+ Longos", "IPCA+ 2040", 10, "Manual (Tesouro Direto)", "=C2*D2", "=E2/SUM(E$2:E$20)", "IPCA + 7.22%", "=TODAY()"},
		{"Tesouro Selic ou Pos-Fixados", "Selic 2031", 5000, "Manual (Tesouro site)", "=C3*D3", "=E3/SUM(E$2:E$20)", "SELIC + 0.1%", "=TODAY()"},
		{"Ouro via ETF", "GOLD11", 100, `=GOOGLEFINANCE("BVMF:GOLD11","price")`, "=C4*D4", "=E4/SUM(E$2:E$20)", "-", "=TODAY()"},
		{"Total Portfolio", "", "", "", "=SUM(E2:E20)", "100%", "", ""},
	})
	_, err = svc.UpdateValues(*spreadsheetID, "Resumo!A1:H5", resumoJSON)
	if err != nil {
		log.Fatalf("Update Resumo: %v", err)
	}
	fmt.Println("Wrote Resumo!A1:H5")

	// Clear old content below Resumo on the first sheet (it was all in one sheet)
	_, _ = svc.ClearValues(*spreadsheetID, "Resumo!A6:Z100")
	fmt.Println("Cleared Resumo!A6:Z100")

	// Transacoes
	transJSON := mustMarshal([][]interface{}{
		{"Data", "Tipo (Compra/Venda)", "Categoria", "Ativo/Ticker", "Quantidade", "Preco Unitario", "Taxas", "Valor Total", "Notas"},
		{"2026-02-01", "Compra", "Tesouro IPCA+ Longos", "IPCA+ 2040", 10, 5000, 0, "=E2*F2-G2", "Inicial"},
	})
	_, err = svc.UpdateValues(*spreadsheetID, "Transacoes!A1:I2", transJSON)
	if err != nil {
		log.Fatalf("Update Transacoes: %v", err)
	}
	fmt.Println("Wrote Transacoes!A1:I2")

	// Ativos
	ativosJSON := mustMarshal([][]interface{}{
		{"Categoria", "Ativo/Ticker", "Descricao", "Yield Atual", "Risco (Baixo/Medio)", "Fonte de Dados"},
		{"Tesouro IPCA+ Longos", "IPCA+ 2040", "Tesouro indexado a inflacao, maturidade 2040", "IPCA + 7.22%", "Baixo", "Tesouro Direto"},
		{"Ouro via ETF", "GOLD11", "ETF de ouro B3", "-", "Medio", "B3"},
	})
	_, err = svc.UpdateValues(*spreadsheetID, "Ativos!A1:F3", ativosJSON)
	if err != nil {
		log.Fatalf("Update Ativos: %v", err)
	}
	fmt.Println("Wrote Ativos!A1:F3")

	// Historico
	histJSON := mustMarshal([][]interface{}{
		{"Data", "Total Portfolio (BRL)", "Alocacao Tesouro %", "Retorno Mensal %", "Inflacao (IPCA)", "Notas"},
		{"2026-02-01", "=Resumo!E5", "Copie de Resumo", "(Atual - Anterior)/Anterior", "Manual (IBGE)", ""},
	})
	_, err = svc.UpdateValues(*spreadsheetID, "Historico!A1:F2", histJSON)
	if err != nil {
		log.Fatalf("Update Historico: %v", err)
	}
	fmt.Println("Wrote Historico!A1:F2")

	fmt.Println("Done. Open:", sp.Properties.Title, "— tabs: Resumo, Transacoes, Ativos, Historico")
}

func mustMarshal(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}
