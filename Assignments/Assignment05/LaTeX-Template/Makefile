TEX = xelatex
TEXFLAGS = -interaction=nonstopmode -halt-on-error
TARGET = HW

# three passes: 1) collect labels 2) settle the problem list 3) fix \pageref{LastPage}
all:
	$(TEX) $(TEXFLAGS) $(TARGET).tex
	$(TEX) $(TEXFLAGS) $(TARGET).tex
	$(TEX) $(TEXFLAGS) $(TARGET).tex
	rm -f *.aux *.log *.lop *.out *.gz

clean: 
	rm -f *.aux *.log *.lop *.out *.gz
	
.PHONY: all clean